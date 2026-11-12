package lddata_test

import (
	lddata "ld-backend/internal/impl/core/data"
	"ld-backend/internal/impl/core/utils"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

func seededFakeData() *lddata.FakeData {
	return lddata.NewFakeData(rand.New(rand.NewPCG(3, 5)))
}

func TestFakeDataSeedRepeatsValues(t *testing.T) {
	a, b := seededFakeData(), seededFakeData()
	for range 20 {
		if a.Id(20) != b.Id(20) || a.FirstName() != b.FirstName() || a.Phone(lddata.Cities[0]) != b.Phone(lddata.Cities[0]) {
			t.Fatal("the same seed must create the same values")
		}
	}
}

func TestNewRandomFakeData(t *testing.T) {
	if lddata.NewRandomFakeData().Id(20) == lddata.NewRandomFakeData().Id(20) {
		t.Fatal("every random fake data must create different values")
	}
}

func TestFakeDataId(t *testing.T) {
	f := seededFakeData()
	valid := regexp.MustCompile(`^[A-Za-z0-9]{20}$`)
	seen := map[string]bool{}
	for range 1000 {
		id := f.Id(20)
		if !valid.MatchString(id) {
			t.Fatalf("invalid id %q", id)
		}
		if seen[id] {
			t.Fatalf("repeated id %q", id)
		}
		seen[id] = true
	}
	if f.Id(0) != "" {
		t.Fatal("an id of length 0 must be empty")
	}
}

func TestFakeDataHex(t *testing.T) {
	value := seededFakeData().Hex(32)
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(value) {
		t.Fatalf("invalid hex %q", value)
	}
}

func TestFakeDataUUID(t *testing.T) {
	f := seededFakeData()
	valid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for range 100 {
		if id := f.UUID(); !valid.MatchString(id) {
			t.Fatalf("invalid version 4 UUID %q", id)
		}
	}
}

func TestFakeDataChance(t *testing.T) {
	f := seededFakeData()
	hits := 0
	for range 10000 {
		if f.Chance(0.25) {
			hits++
		}
	}
	if hits < 2200 || hits > 2800 {
		t.Fatalf("expected about 2500 hits of 25%%, got %d", hits)
	}
	if f.Chance(0) || !f.Chance(1) {
		t.Fatal("chance 0 must never hit and chance 1 must always hit")
	}
}

func TestFakeDataBetween(t *testing.T) {
	f := seededFakeData()
	for range 1000 {
		d := f.Between(time.Hour, 2*time.Hour)
		if d < time.Hour || d >= 2*time.Hour {
			t.Fatalf("duration %v out of range", d)
		}
	}
	if f.Between(time.Hour, time.Hour) != time.Hour || f.Between(2*time.Hour, time.Hour) != 2*time.Hour {
		t.Fatal("an empty range must give its low end")
	}
}

func TestFakeDataDistinct(t *testing.T) {
	f := seededFakeData()

	names := f.Distinct(20, f.FirstName)
	if len(names) != 20 {
		t.Fatalf("expected 20 names, got %d", len(names))
	}
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			t.Fatalf("repeated name %q", name)
		}
		seen[name] = true
	}

	// a function that always repeats still gives the count
	same := f.Distinct(3, func() string { return "Acme" })
	if !slices.Equal(same, []string{"Acme", "Acme 2", "Acme 3"}) {
		t.Fatalf("unexpected values %v", same)
	}

	empty := f.Distinct(2, func() string { return "" })
	if !slices.Equal(empty, []string{"1", "2"}) {
		t.Fatalf("unexpected values %v", empty)
	}
}

func TestFakeDataPhone(t *testing.T) {
	f := seededFakeData()
	city := lddata.Cities[0]
	seen := map[string]bool{}
	for range 2000 {
		phone := f.Phone(city)
		number, err := lddata.ValidatePhoneNumber(phone)
		if err != nil {
			t.Fatalf("invalid phone %q: %v", phone, err)
		}
		if !slices.Contains(city.AreaCodes, number[1:4]) {
			t.Fatalf("phone %q is not in an area code of %s", phone, city.Name)
		}
		if number[4] < '2' {
			t.Fatalf("phone %q has an invalid exchange", phone)
		}
		if seen[phone] {
			t.Fatalf("repeated phone %q", phone)
		}
		seen[phone] = true
	}
}

func TestFakeDataAddresses(t *testing.T) {
	f := seededFakeData()
	for range 100 {
		city := f.City()
		if !regexp.MustCompile(`^\d{3,4} \w+ \w+$`).MatchString(f.Street()) {
			t.Fatal("invalid street")
		}
		zip := f.PostalCode(city)
		if len(zip) != 5 || !strings.HasPrefix(zip, city.ZipPrefix) {
			t.Fatalf("invalid ZIP %q for %s", zip, city.Name)
		}
		if _, err := time.LoadLocation(city.Timezone); err != nil {
			t.Fatalf("invalid time zone %q: %v", city.Timezone, err)
		}
	}
}

func TestFakeDataEmails(t *testing.T) {
	f := seededFakeData()
	personal := regexp.MustCompile(`^mary\.lee\d{0,2}@[a-z]+\.com$`)
	for range 100 {
		email := f.PersonalEmail("Mary", "Lee")
		if !personal.MatchString(email) {
			t.Fatalf("invalid personal email %q", email)
		}
	}
	if email := f.WorkEmail("Mary", "Lee", "summitgrowth.com"); email != "mary.lee@summitgrowth.com" {
		t.Fatalf("unexpected work email %q", email)
	}
}

func TestFakeDataBusinesses(t *testing.T) {
	f := seededFakeData()
	for range 200 {
		agency := f.AgencyName()
		if len(strings.Fields(agency)) < 2 {
			t.Fatalf("agency name %q is too short", agency)
		}
		city := f.City()
		business := f.BusinessName(city)
		if strings.Contains(business, "{") {
			t.Fatalf("business name %q has placeholders", business)
		}
		company := f.CompanyName("Lee")
		if !strings.HasPrefix(company, "Lee ") {
			t.Fatalf("unexpected company name %q", company)
		}
		if !slices.Contains(lddata.LeadSources, f.LeadSource()) {
			t.Fatal("unknown lead source")
		}
	}
}

func TestFakeDataTags(t *testing.T) {
	f := seededFakeData()
	sizes := map[int]bool{}
	for range 500 {
		tags := f.Tags(3)
		sizes[len(tags)] = true
		if len(tags) > 3 {
			t.Fatalf("too many tags %v", tags)
		}
		seen := map[string]bool{}
		for _, tag := range tags {
			if seen[tag] || !slices.Contains(lddata.LeadTags, tag) {
				t.Fatalf("invalid tags %v", tags)
			}
			seen[tag] = true
		}
	}
	for size := range 4 {
		if !sizes[size] {
			t.Fatalf("no list of %d tags was created", size)
		}
	}
	if len(f.Tags(100)) > len(lddata.LeadTags) {
		t.Fatal("tags are limited by the known tags")
	}
}

func TestPick(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	seen := map[string]bool{}
	for range 300 {
		seen[lddata.Pick(rng, []string{"a", "b", "c"})] = true
	}
	if len(seen) != 3 {
		t.Fatalf("every value must be picked, got %v", seen)
	}
}

func TestPickWeighted(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	values := []lddata.Weighted[string]{{Value: "often", Weight: 9}, {Value: "rarely", Weight: 1}, {Value: "never", Weight: 0}}
	counts := map[string]int{}
	for range 10000 {
		counts[lddata.PickWeighted(rng, values)]++
	}
	if counts["never"] != 0 {
		t.Fatal("a value without weight must not be picked")
	}
	if counts["often"] < 8700 || counts["often"] > 9300 {
		t.Fatalf("expected about 9000 picks of the 90%% value, got %d", counts["often"])
	}
}

func TestDomainOf(t *testing.T) {
	tests := map[string]string{
		"Summit Growth Partners": "summitgrowthpartners.com",
		"Bold Arrow Media Co":    "boldarrowmediaco.com",
		"A&B 24/7 Services!":     "ab247services.com",
	}
	for name, expected := range tests {
		if got := lddata.DomainOf(name); got != expected {
			t.Fatalf("DomainOf(%q) = %q, want %q", name, got, expected)
		}
	}
}

func TestRealisticValues(t *testing.T) {
	lists := map[string][]string{
		"first names": lddata.FirstNames, "last names": lddata.LastNames, "street names": lddata.StreetNames,
		"agency prefixes": lddata.AgencyPrefixes, "business templates": lddata.BusinessTemplates, "lead tags": lddata.LeadTags,
	}
	for name, values := range lists {
		seen := map[string]bool{}
		for _, value := range values {
			if seen[value] {
				t.Fatalf("%s repeat %q", name, value)
			}
			seen[value] = true
		}
	}
	for _, city := range lddata.Cities {
		if len(city.State) != 2 || len(city.ZipPrefix) != 3 || len(city.AreaCodes) == 0 {
			t.Fatalf("invalid city %+v", city)
		}
	}
}

func TestPtr(t *testing.T) {
	value := []string{"a"}
	ptr := ldutils.Ptr(value)
	if len(*ptr) != 1 || (*ptr)[0] != "a" {
		t.Fatal("unexpected value")
	}
	n := 3
	p := ldutils.Ptr(n)
	n = 4
	if *p != 3 {
		t.Fatal("the pointer must be to a copy")
	}
}
