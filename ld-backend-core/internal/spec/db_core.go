package ldspec

import (
	"time"

	"gorm.io/gorm"
)

// DB_SCHEMA_VERSION is the version of the core and tenant schemas; increase
// it when a table changes so the databases are migrated again.
const DB_SCHEMA_VERSION = "1.0.3"

// DbSchemaVersion is a migration of the database, kept in the core and tenant
// databases; the latest one tells the version of their schema.
type DbSchemaVersion struct {
	gorm.Model
	// Version is the DB_SCHEMA_VERSION the database was migrated to.
	Version string `gorm:"type:varchar(20)"`
}

// TableName is the table of the schema versions.
func (DbSchemaVersion) TableName() string {
	return "schema_versions"
}

// LDActiveStatus is the stored status of a record that can be disabled.
type LDActiveStatus string

const (
	// LDActive is an enabled record.
	LDActive LDActiveStatus = "actv"
	// LDInactive is a disabled record.
	LDInactive LDActiveStatus = "inac"
)

type LDSubscriptionStatus string

const (
	LDSubscriptionActive   LDSubscriptionStatus = "active"
	LDSubscriptionCanceled LDSubscriptionStatus = "cancelled"
	LDSubscriptionUnPaid   LDSubscriptionStatus = "unpaid"
)

type LDServerNodeStatus string

const (
	Starting    LDServerNodeStatus = "starting"
	Live        LDServerNodeStatus = "live"
	ShutingDown LDServerNodeStatus = "shutingdown"
	Down        LDServerNodeStatus = "down"
)

type LDUserAccess string

const (
	// Dialer can dial
	LDUserAccessDialer LDUserAccess = "dialer"
	// Supervisor can manage dialers, contacts, and pipelines and see the dashboard
	LDUserAccessSupervisor LDUserAccess = "supervisor"
	// Admin can manage dialers, supervisors, contacts, pipelines, and tenants
	LDUserAccessAdmin LDUserAccess = "admin"
	// Owner can manage dialers, supervisors, contacts, pipelines, tenants, and service accounts, it can also call
	LDUserAccessOwner LDUserAccess = "owner"
)

// IsValid tells if the access is one of the user accesses.
func (a LDUserAccess) IsValid() bool {
	switch a {
	case LDUserAccessDialer, LDUserAccessSupervisor, LDUserAccessAdmin, LDUserAccessOwner:
		return true
	}
	return false
}

type DbPaidSubscription struct {
	gorm.Model
	BusinessId       string
	LocationId       string `gorm:"uniqueIndex"`
	StripeCustomerId *string
	Status           LDSubscriptionStatus `gorm:"type:varchar(15);index;"`
	MonthlyAmount    *float64
	PerUserAmount    *float64
	AmountDue        *float64
	Accounts         *uint
	LastPayment      *time.Time
}

func (DbPaidSubscription) TableName() string {
	return "paid_subscriptions"
}

type DbPaymentRecord struct {
	gorm.Model
	BusinessId *string
	LocationId *string
	StripeId   *string
	Amount     *float64
	DateTime   *time.Time
}

func (DbPaymentRecord) TableName() string {
	return "payment_records"
}

// DbServiceAccount holds the client credentials an external service uses to
// call the LeadDial API.
type DbServiceAccount struct {
	gorm.Model
	ClientId     string `gorm:"uniqueIndex"`
	ClientSecret *string
	Status       LDActiveStatus `gorm:"type:char(4);index;default:'actv'"`
}

func (DbServiceAccount) TableName() string {
	return "service_accounts"
}

type DbServerNode struct {
	gorm.Model
	NodeId        string             `gorm:"uniqueIndex"`
	Status        LDServerNodeStatus `gorm:"type:varchar(15);index;default:'down'"`
	LastHeartbeat *time.Time         `gorm:"index"`
}

func (DbServerNode) TableName() string {
	return "server_nodes"
}

// DbSystemAdmin is an administrator of the whole system, kept in the core
// database.
type DbSystemAdmin struct {
	gorm.Model
	// Username is the unique login name.
	Username string `gorm:"uniqueIndex"`
	// Password is the stored password of the admin.
	Password *string
	// Status tells whether the admin can log in.
	Status LDActiveStatus `gorm:"type:char(4);index;default:'actv'"`
}

// TableName is the table of the system admins.
func (DbSystemAdmin) TableName() string {
	return "system_admins"
}

// DbTenant is a GHL location with its own database, kept in the core
// database.
type DbTenant struct {
	gorm.Model
	// BusinessId is the GHL location id the tenant belongs to.
	BusinessId string `gorm:"type:varchar(50);index"`
	// DbName is the name of the tenant database, the lowercase BusinessId.
	DbName string `gorm:"type:varchar(50);index"`
	// Title is the display name of the tenant.
	Title string `gorm:"type:varchar(100)"`
	// Url is the connection url of the tenant database.
	Url string `gorm:"type:varchar(255)"`
	// Driver is the LDDatabaseDriver of the tenant database.
	Driver string `gorm:"type:varchar(20);index"`
	// Status tells whether the tenant is enabled.
	Status LDActiveStatus `gorm:"type:char(4);index;default:'actv'"`
}

// TableName is the table of the tenants.
func (DbTenant) TableName() string {
	return "tenants"
}

type DbGHLCompany struct {
	gorm.Model
	CompanyId       string `gorm:"uniqueIndex"`
	Name            string
	InstallAuthCode *string `gorm:"uniqueIndex"`
	AccessToken     *string
	RefreshToken    *string
	UserId          *string
	// ExpiresIn is the lifetime in seconds of the AccessToken, counted from
	// UpdatedAt.
	ExpiresIn *int64
}

func (DbGHLCompany) TableName() string {
	return "ghl_companies"
}

type DbGHLInstallType string

const (
	DbGHLInstallTypeCompany  = "company"
	DbGHLInstallTypeLocation = "location"
)

type DbGHLLocation struct {
	gorm.Model
	CompanyId       string
	LocationId      string `gorm:"uniqueIndex"`
	Name            string
	AccessToken     *string
	RefreshToken    *string
	InstallType     DbGHLInstallType
	UserId          *string
	InstallAuthCode *string `gorm:"uniqueIndex"`
	// ExpiresIn is the lifetime in seconds of the AccessToken, counted from
	// UpdatedAt.
	ExpiresIn *int64
}

func (DbGHLLocation) TableName() string {
	return "ghl_locations"
}

// coreTables are the tables of the core database.
var DbCoreTables = []interface{}{
	DbSystemAdmin{},
	DbTenant{},
	DbGHLCompany{},
	DbGHLLocation{},
	DbServiceAccount{},
	DbServerNode{},
	DbPaidSubscription{},
	DbSchemaVersion{},
	DbPaymentRecord{},
}

// CoreRepo is the repository of the core database, with the tenants and the
// GHL installations.
type CoreRepo interface {
	// FindTenant finds the tenant of the GHL location id.
	FindTenant(id string) (*DbTenant, error)
	// SaveTenant creates or updates the tenant.
	SaveTenant(result *DbTenant) error

	FindByAuthCode(authCode string) (*DbGHLCompany, *DbGHLLocation, error)
	FindCompanies() ([]*DbGHLCompany, error)
	FindAllLocations() ([]*DbGHLLocation, error)
	FindCompany(companyId string) (*DbGHLCompany, error)
	SaveCompanyAppInstall(authCode string, company *DbGHLCompany) (*DbGHLCompany, error)
	SaveCompany(company *DbGHLCompany) error
	FindLocations(companyId string) ([]*DbGHLLocation, error)
	FindLocation(locationId string) (*DbGHLLocation, error)
	SaveLocation(location *DbGHLLocation) error
	SaveLocationAppInstall(authCode string, location *DbGHLLocation) (*DbGHLLocation, error)

	FindServiceAccount(clientId *string) (*DbServiceAccount, error)
	SaveServiceAccount(account *DbServiceAccount) error

	FindNode(nodeId string) (*DbServerNode, error)
	FindAllNodes() ([]*DbServerNode, error)
	SaveNode(node *DbServerNode) error

	FindPayments(locationId string) ([]*DbPaymentRecord, error)
	SavePayment(payment *DbPaymentRecord) error

	FindPaidSubscription(locationId string) (*DbPaidSubscription, error)
	SavePaidSubscription(paidSubscription *DbPaidSubscription) error
}
