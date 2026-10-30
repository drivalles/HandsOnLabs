package core

import (
	"errors"
	"fmt"
	"ld-backend/internal/impl/back/db/core/query"
	"ld-backend/internal/impl/utils"
	"ld-backend/internal/spec"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	// activityDuration is how long the activity runs on each driver.
	activityDuration = 10 * time.Minute
	activityWorkers  = 8
	activityTenants  = 20
	activityAccounts = 10
	activityAdmins   = 5
	activityNodes    = 5
	activityCompany  = 10
	activityLocation = 50
)

// TestActivity simulates the real activity of the servers on the core
// database: several workers read and write every table at the same time,
// with a few keys so they update the same records. It runs for 10 minutes on
// all the drivers at once, so it only runs when LD_ACTIVITY_TEST is set;
// LD_ACTIVITY_DURATION changes how long it runs, e.g. LD_ACTIVITY_DURATION=30s.
func TestActivity(t *testing.T) {
	if os.Getenv("LD_ACTIVITY_TEST") == "" {
		t.Skip("set LD_ACTIVITY_TEST=1 to run the activity on the core database")
	}
	duration := activityDuration
	if value := os.Getenv("LD_ACTIVITY_DURATION"); value != "" {
		parsed, err := time.ParseDuration(value)
		require.NoError(t, err)
		duration = parsed
	}

	for _, d := range TestDrivers() {
		t.Run(d.Name, func(t *testing.T) {
			t.Parallel()
			d.Check(t)
			if d.Cfg.Driver == spec.SqliteDriver && d.Cfg.InMemorySQLite {
				t.Skip("the shared cache of in-memory sqlite fails concurrent writes with table locked errors")
			}
			a := newActivity(t, newTestRepo(t, d))
			a.seed()
			a.run(duration)
			a.check()
		})
	}
}

// activity is the state of a simulation on one database.
type activity struct {
	t    *testing.T
	repo *coreRepoImpl
	q    *query.Query
	ops  []activityOp

	mu     sync.Mutex
	counts map[string]int
	errs   map[string]int
	// codes are the latest authorization codes of each company.
	codes map[string]string
	// nodeMu saves the heartbeats of a node one at a time, as the node does.
	nodeMu sync.Mutex
	// heartbeats are the latest heartbeats saved of each node.
	heartbeats map[string]time.Time
}

// activityOp is an operation the workers run, picked by its weight.
type activityOp struct {
	name   string
	weight int
	run    func(r *rand.Rand) error
}

func newActivity(t *testing.T, repo *coreRepoImpl) *activity {
	a := &activity{
		t:          t,
		repo:       repo,
		q:          query.Use(repo.coreDb),
		counts:     map[string]int{},
		errs:       map[string]int{},
		codes:      map[string]string{},
		heartbeats: map[string]time.Time{},
	}
	a.ops = []activityOp{
		{"SaveSystemAdmin", 2, a.saveSystemAdmin},
		{"FindSystemAdmin", 4, a.findSystemAdmin},
		{"SaveTenant", 3, a.saveTenant},
		{"FindTenant", 10, a.findTenant},
		{"SaveAppInstall", 6, a.saveAppInstall},
		{"GetLatestInstall", 6, a.getLatestInstall},
		{"GetLatestAccess", 8, a.getLatestAccess},
		{"FindByCompanyAndAuthCode", 3, a.findByCompanyAndAuthCode},
		{"FindByAuthCode", 3, a.findByAuthCode},
		{"FindCompanies", 2, a.findCompanies},
		{"FindAllAppInstalls", 1, a.findAllAppInstalls},
		{"CleanAppInstalls", 1, a.cleanAppInstalls},
		{"SaveInstalledLocation", 4, a.saveInstalledLocation},
		{"FindInstalledLocation", 6, a.findInstalledLocation},
		{"FindInstalledLocations", 3, a.findInstalledLocations},
		{"SaveServiceAccount", 2, a.saveServiceAccount},
		{"FindServiceAccount", 6, a.findServiceAccount},
		{"SaveNode", 8, a.saveNode},
		{"FindNode", 4, a.findNode},
		{"FindAllNodes", 4, a.findAllNodes},
	}
	return a
}

// seed creates the records the workers update; the tenants, accounts and
// admins are created once by the servers and only updated later.
func (a *activity) seed() {
	for i := 0; i < activityTenants; i++ {
		require.NoError(a.t, a.repo.SaveTenant(&spec.DbTenant{
			BusinessId: activityLocationId(i),
			DbName:     activityLocationId(i),
			Title:      "Tenant",
		}))
	}
	for i := 0; i < activityAccounts; i++ {
		require.NoError(a.t, a.repo.SaveServiceAccount(&spec.DbServiceAccount{ClientId: activityClientId(i)}))
	}
	for i := 0; i < activityAdmins; i++ {
		require.NoError(a.t, a.q.DbSystemAdmin.Create(&spec.DbSystemAdmin{Username: activityAdmin(i)}))
	}
}

// run runs the workers until the duration ends, logging the progress each
// minute.
func (a *activity) run(duration time.Duration) {
	deadline := time.Now().Add(duration)
	done := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < activityWorkers; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			for time.Now().Before(deadline) {
				a.runOp(r)
				time.Sleep(time.Duration(5+r.Intn(20)) * time.Millisecond)
			}
		}(time.Now().UnixNano() + int64(w))
	}
	go func() {
		wg.Wait()
		close(done)
	}()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			a.logStats(duration)
			return
		case <-ticker.C:
			total, errs := a.totals()
			a.t.Logf("%d operations, %d errors, %s left", total, errs, time.Until(deadline).Round(time.Second))
		}
	}
}

func (a *activity) runOp(r *rand.Rand) {
	total := 0
	for _, op := range a.ops {
		total += op.weight
	}
	pick := r.Intn(total)
	for _, op := range a.ops {
		if pick >= op.weight {
			pick -= op.weight
			continue
		}
		err := op.run(r)
		a.mu.Lock()
		defer a.mu.Unlock()
		a.counts[op.name]++
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			a.errs[fmt.Sprintf("%s: %v", op.name, err)]++
		}
		return
	}
}

// check verifies the records are consistent after the activity.
func (a *activity) check() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nodeMu.Lock()
	defer a.nodeMu.Unlock()
	for err, count := range a.errs {
		a.t.Errorf("%d times %s", count, err)
	}

	count := func(model interface{}) int64 {
		var n int64
		require.NoError(a.t, a.repo.coreDb.Model(model).Count(&n).Error)
		return n
	}
	assert.EqualValues(a.t, activityTenants, count(&spec.DbTenant{}))
	assert.EqualValues(a.t, activityAccounts, count(&spec.DbServiceAccount{}))
	assert.EqualValues(a.t, activityAdmins, count(&spec.DbSystemAdmin{}))
	assert.LessOrEqual(a.t, count(&spec.DbServerNode{}), int64(activityNodes))
	assert.LessOrEqual(a.t, count(&spec.DbGHLInstalledLocation{}), int64(activityLocation))

	// the nodes keep the latest heartbeat saved
	nodes, err := a.repo.FindAllNodes()
	require.NoError(a.t, err)
	assert.Len(a.t, nodes, len(a.heartbeats))
	for _, node := range nodes {
		require.NotNil(a.t, node.LastHeartbeat, node.NodeId)
		assert.WithinDuration(a.t, a.heartbeats[node.NodeId], *node.LastHeartbeat, time.Microsecond, node.NodeId)
	}
}

func (a *activity) totals() (int, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	total, errs := 0, 0
	for _, n := range a.counts {
		total += n
	}
	for _, n := range a.errs {
		errs += n
	}
	return total, errs
}

func (a *activity) logStats(duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	names := make([]string, 0, len(a.counts))
	total := 0
	for name, n := range a.counts {
		names = append(names, name)
		total += n
	}
	sort.Strings(names)
	for _, name := range names {
		a.t.Logf("%-25s %d", name, a.counts[name])
	}
	a.t.Logf("%d operations in %s, %.0f per second", total, duration, float64(total)/duration.Seconds())
}

func activityLocationId(i int) string { return fmt.Sprintf("loc-%d", i) }
func activityCompanyId(i int) string  { return fmt.Sprintf("company-%d", i%activityCompany) }
func activityClientId(i int) string   { return fmt.Sprintf("client-%d", i) }
func activityAdmin(i int) string      { return fmt.Sprintf("admin-%d", i) }
func activityNodeId(i int) string     { return fmt.Sprintf("node-%d", i) }

func (a *activity) saveSystemAdmin(r *rand.Rand) error {
	admin, err := a.findAdmin(activityAdmin(r.Intn(activityAdmins)))
	if err != nil {
		return err
	}
	admin.Password = utils.Ptr(fmt.Sprintf("password-%d", r.Int()))
	return a.q.DbSystemAdmin.Save(admin)
}

func (a *activity) findSystemAdmin(r *rand.Rand) error {
	_, err := a.findAdmin(activityAdmin(r.Intn(activityAdmins)))
	return err
}

func (a *activity) findAdmin(username string) (*spec.DbSystemAdmin, error) {
	return a.q.DbSystemAdmin.Where(a.q.DbSystemAdmin.Username.Eq(username)).First()
}

func (a *activity) saveTenant(r *rand.Rand) error {
	tenant, err := a.repo.FindTenant(activityLocationId(r.Intn(activityTenants)))
	if err != nil {
		return err
	}
	tenant.Title = fmt.Sprintf("Tenant %d", r.Int())
	tenant.Status = spec.LDActive
	if r.Intn(5) == 0 {
		tenant.Status = spec.LDInactive
	}
	return a.repo.SaveTenant(tenant)
}

func (a *activity) findTenant(r *rand.Rand) error {
	_, err := a.repo.FindTenant(activityLocationId(r.Intn(activityTenants)))
	return err
}

// saveAppInstall saves a new company grant, or a location token, as the GHL
// app does when installed or when its tokens are refreshed.
func (a *activity) saveAppInstall(r *rand.Rand) error {
	location := r.Intn(activityLocation)
	companyId := activityCompanyId(location)
	code := fmt.Sprintf("code-%d", r.Int())
	if r.Intn(3) == 0 {
		if err := a.repo.SaveAppInstall(CompanyGrant(companyId, code, time.Now())); err != nil {
			return err
		}
		a.mu.Lock()
		a.codes[companyId] = code
		a.mu.Unlock()
		return nil
	}
	return a.repo.SaveAppInstall(LocationAccess(companyId, activityLocationId(location), code, time.Now()))
}

func (a *activity) getLatestInstall(r *rand.Rand) error {
	_, err := a.repo.GetLatestInstall(activityCompanyId(r.Intn(activityCompany)))
	return err
}

func (a *activity) getLatestAccess(r *rand.Rand) error {
	_, err := a.repo.GetLatestAccess(activityLocationId(r.Intn(activityLocation)))
	return err
}

func (a *activity) latestCode(r *rand.Rand) (string, string) {
	companyId := activityCompanyId(r.Intn(activityCompany))
	a.mu.Lock()
	defer a.mu.Unlock()
	return companyId, a.codes[companyId]
}

func (a *activity) findByCompanyAndAuthCode(r *rand.Rand) error {
	companyId, code := a.latestCode(r)
	_, err := a.repo.FindByCompanyAndAuthCode(companyId, code)
	return err
}

func (a *activity) findByAuthCode(r *rand.Rand) error {
	_, code := a.latestCode(r)
	_, err := a.repo.FindByAuthCode(code)
	return err
}

func (a *activity) findCompanies(_ *rand.Rand) error {
	_, err := a.repo.FindCompanies()
	return err
}

func (a *activity) findAllAppInstalls(_ *rand.Rand) error {
	_, err := a.repo.FindAllAppInstalls()
	return err
}

// cleanAppInstalls deletes the installations older than a minute, as the
// servers clean the expired tokens.
func (a *activity) cleanAppInstalls(_ *rand.Rand) error {
	return a.repo.CleanAppInstalls(time.Now().Add(-time.Minute))
}

// saveInstalledLocation saves the location as a new record, as when the
// locations of a company are imported from GHL again.
func (a *activity) saveInstalledLocation(r *rand.Rand) error {
	location := r.Intn(activityLocation)
	installedAt := time.Now()
	return a.repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{
		CompanyId:   activityCompanyId(location),
		LocationId:  activityLocationId(location),
		Name:        fmt.Sprintf("Location %d", r.Int()),
		IsInstalled: utils.Ptr(r.Intn(4) != 0),
		InstalledAt: &installedAt,
	})
}

func (a *activity) findInstalledLocation(r *rand.Rand) error {
	_, err := a.repo.FindInstalledLocation(activityLocationId(r.Intn(activityLocation)))
	return err
}

func (a *activity) findInstalledLocations(r *rand.Rand) error {
	_, err := a.repo.FindInstalledLocations(activityCompanyId(r.Intn(activityCompany)))
	return err
}

func (a *activity) saveServiceAccount(r *rand.Rand) error {
	account, err := a.repo.FindServiceAccount(utils.Ptr(activityClientId(r.Intn(activityAccounts))))
	if err != nil {
		return err
	}
	account.ClientSecret = utils.Ptr(fmt.Sprintf("secret-%d", r.Int()))
	return a.repo.SaveServiceAccount(account)
}

func (a *activity) findServiceAccount(r *rand.Rand) error {
	_, err := a.repo.FindServiceAccount(utils.Ptr(activityClientId(r.Intn(activityAccounts))))
	return err
}

// saveNode saves a heartbeat of the node as a new record, as the servers do
// while running.
func (a *activity) saveNode(r *rand.Rand) error {
	nodeId := activityNodeId(r.Intn(activityNodes))
	a.nodeMu.Lock()
	defer a.nodeMu.Unlock()
	heartbeat := time.Now().Truncate(time.Microsecond)
	err := a.repo.SaveNode(&spec.DbServerNode{NodeId: nodeId, Status: spec.Live, LastHeartbeat: &heartbeat})
	if err == nil {
		a.heartbeats[nodeId] = heartbeat
	}
	return err
}

func (a *activity) findNode(r *rand.Rand) error {
	_, err := a.repo.FindNode(activityNodeId(r.Intn(activityNodes)))
	return err
}

func (a *activity) findAllNodes(_ *rand.Rand) error {
	_, err := a.repo.FindAllNodes()
	return err
}
