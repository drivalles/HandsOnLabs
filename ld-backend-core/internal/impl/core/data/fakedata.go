package lddata

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

// FakeData creates realistic random values, as people, places, phones and
// business names, from its random source; the phones it creates are unique.
type FakeData struct {
	rng        *rand.Rand
	usedPhones map[string]bool
}

// NewFakeData creates the values from the random source, so a seeded source
// repeats them.
func NewFakeData(rng *rand.Rand) *FakeData {
	return &FakeData{rng: rng, usedPhones: map[string]bool{}}
}

// NewRandomFakeData creates different values on every run.
func NewRandomFakeData() *FakeData {
	return NewFakeData(rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
}

// Rand is the random source of the values.
func (f *FakeData) Rand() *rand.Rand {
	return f.rng
}

const idAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// Id is a random id of letters and digits of the length.
func (f *FakeData) Id(length int) string {
	var b strings.Builder
	for range length {
		b.WriteByte(idAlphabet[f.rng.IntN(len(idAlphabet))])
	}
	return b.String()
}

// Hex is a random value of the length in hexadecimal digits.
func (f *FakeData) Hex(length int) string {
	var b strings.Builder
	for range length {
		b.WriteByte("0123456789abcdef"[f.rng.IntN(16)])
	}
	return b.String()
}

// UUID is a random version 4 UUID.
func (f *FakeData) UUID() string {
	a, b := f.rng.Uint64(), f.rng.Uint64()
	return fmt.Sprintf("%08x-%04x-4%03x-%04x-%012x", a>>32, (a>>16)&0xffff, a&0xfff, 0x8000|(b>>48)&0x3fff, b&0xffffffffffff)
}

// Chance is true with the probability, from 0 to 1.
func (f *FakeData) Chance(probability float64) bool {
	return f.rng.Float64() < probability
}

// Between is a random duration from low up to high; low when the range is
// empty.
func (f *FakeData) Between(low time.Duration, high time.Duration) time.Duration {
	if high <= low {
		return low
	}
	return low + time.Duration(f.rng.Int64N(int64(high-low)))
}

// Distinct creates count different values with the function. A repeated
// value is created again; when the function keeps repeating, or gives an
// empty value, the position is added to make it unique.
func (f *FakeData) Distinct(count int, value func() string) []string {
	seen := map[string]bool{}
	result := []string{}
	for attempt := 0; len(result) < count; attempt++ {
		candidate := value()
		if attempt > count*20 || candidate == "" {
			candidate = strings.TrimSpace(fmt.Sprintf("%s %d", candidate, len(result)+1))
		}
		if !seen[candidate] {
			seen[candidate] = true
			result = append(result, candidate)
		}
	}
	return result
}

// ===== people and places =====

// City is a US city with the values of its addresses and phones.
type City struct {
	Name      string
	State     string
	ZipPrefix string
	Timezone  string
	AreaCodes []string
}

func (f *FakeData) City() City {
	return Pick(f.rng, Cities)
}

func (f *FakeData) FirstName() string {
	return Pick(f.rng, FirstNames)
}

func (f *FakeData) LastName() string {
	return Pick(f.rng, LastNames)
}

// Phone is a unique US number in E.164 format, in an area code of the city.
func (f *FakeData) Phone(city City) string {
	for {
		areaCode := Pick(f.rng, city.AreaCodes)
		number := fmt.Sprintf("+1%s%d%02d%04d", areaCode, 2+f.rng.IntN(8), f.rng.IntN(100), f.rng.IntN(10000))
		if !f.usedPhones[number] {
			f.usedPhones[number] = true
			return number
		}
	}
}

// Street is a street address, as 1234 Oak Ave.
func (f *FakeData) Street() string {
	return fmt.Sprintf("%d %s %s", 100+f.rng.IntN(9800), Pick(f.rng, StreetNames), Pick(f.rng, StreetSuffixes))
}

// PostalCode is a ZIP code of the city.
func (f *FakeData) PostalCode(city City) string {
	return fmt.Sprintf("%s%02d", city.ZipPrefix, f.rng.IntN(100))
}

// PersonalEmail is an email of the person at a common provider, sometimes
// with a number.
func (f *FakeData) PersonalEmail(first string, last string) string {
	user := strings.ToLower(first + "." + last)
	if f.Chance(0.4) {
		user += strconv.Itoa(f.rng.IntN(99) + 1)
	}
	return user + "@" + Pick(f.rng, EmailDomains)
}

// WorkEmail is the email of the person at the company domain.
func (f *FakeData) WorkEmail(first string, last string, domain string) string {
	return strings.ToLower(first+"."+last) + "@" + domain
}

// ===== businesses =====

// AgencyName is the name of a marketing agency, as Summit Growth Partners.
func (f *FakeData) AgencyName() string {
	return Pick(f.rng, AgencyPrefixes) + " " + Pick(f.rng, AgencySuffixes)
}

// BusinessName is the name of a local business in the city, as Garcia
// Family Dental.
func (f *FakeData) BusinessName(city City) string {
	name := strings.ReplaceAll(Pick(f.rng, BusinessTemplates), "{last}", f.LastName())
	return strings.ReplaceAll(name, "{city}", city.Name)
}

// CompanyName is the name of a small company of the family, as Lee LLC.
func (f *FakeData) CompanyName(lastName string) string {
	return lastName + " " + Pick(f.rng, SmallBusinessSuffixes)
}

// LeadSource is where a lead came from, as Facebook Ad.
func (f *FakeData) LeadSource() string {
	return Pick(f.rng, LeadSources)
}

// Tags are up to max different lead tags, maybe none.
func (f *FakeData) Tags(max int) []string {
	count := f.rng.IntN(min(max, len(LeadTags)) + 1)
	result := []string{}
	for _, index := range f.rng.Perm(len(LeadTags))[:count] {
		result = append(result, LeadTags[index])
	}
	return result
}

// ===== random choices =====

// Pick is a random value of the list, which must not be empty.
func Pick[T any](rng *rand.Rand, values []T) T {
	return values[rng.IntN(len(values))]
}

// Weighted is a value with its share of the choices of PickWeighted.
type Weighted[T any] struct {
	Value  T
	Weight int
}

// PickWeighted is a random value, chosen in proportion to the weights; the
// list must have a positive weight.
func PickWeighted[T any](rng *rand.Rand, values []Weighted[T]) T {
	total := 0
	for _, v := range values {
		total += v.Weight
	}
	choice := rng.IntN(total)
	for _, v := range values {
		if choice < v.Weight {
			return v.Value
		}
		choice -= v.Weight
	}
	return values[len(values)-1].Value
}

// DomainOf is a .com domain made of the letters and digits of the name.
func DomainOf(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String() + ".com"
}

// ===== realistic values =====

var Cities = []City{
	{"Austin", "TX", "787", "America/Chicago", []string{"512", "737"}},
	{"Dallas", "TX", "752", "America/Chicago", []string{"214", "469", "972"}},
	{"Houston", "TX", "770", "America/Chicago", []string{"713", "281", "832"}},
	{"Phoenix", "AZ", "850", "America/Phoenix", []string{"602", "480", "623"}},
	{"Denver", "CO", "802", "America/Denver", []string{"303", "720"}},
	{"Miami", "FL", "331", "America/New_York", []string{"305", "786"}},
	{"Tampa", "FL", "336", "America/New_York", []string{"813", "656"}},
	{"Orlando", "FL", "328", "America/New_York", []string{"407", "689"}},
	{"Atlanta", "GA", "303", "America/New_York", []string{"404", "470", "678"}},
	{"Charlotte", "NC", "282", "America/New_York", []string{"704", "980"}},
	{"Nashville", "TN", "372", "America/Chicago", []string{"615", "629"}},
	{"Chicago", "IL", "606", "America/Chicago", []string{"312", "773", "872"}},
	{"Columbus", "OH", "432", "America/New_York", []string{"614", "380"}},
	{"Seattle", "WA", "981", "America/Los_Angeles", []string{"206", "253"}},
	{"Portland", "OR", "972", "America/Los_Angeles", []string{"503", "971"}},
	{"San Diego", "CA", "921", "America/Los_Angeles", []string{"619", "858"}},
	{"Sacramento", "CA", "958", "America/Los_Angeles", []string{"916", "279"}},
	{"Las Vegas", "NV", "891", "America/Los_Angeles", []string{"702", "725"}},
	{"Salt Lake City", "UT", "841", "America/Denver", []string{"801", "385"}},
	{"Boston", "MA", "021", "America/New_York", []string{"617", "857"}},
	{"Philadelphia", "PA", "191", "America/New_York", []string{"215", "267"}},
	{"Minneapolis", "MN", "554", "America/Chicago", []string{"612", "952"}},
	{"Kansas City", "MO", "641", "America/Chicago", []string{"816", "975"}},
	{"Raleigh", "NC", "276", "America/New_York", []string{"919", "984"}},
}

var FirstNames = []string{
	"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda", "David", "Elizabeth",
	"William", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Charles", "Karen",
	"Christopher", "Lisa", "Daniel", "Nancy", "Matthew", "Betty", "Anthony", "Sandra", "Mark", "Margaret",
	"Donald", "Ashley", "Steven", "Kimberly", "Andrew", "Emily", "Paul", "Donna", "Joshua", "Michelle",
	"Kevin", "Carol", "Brian", "Amanda", "George", "Melissa", "Timothy", "Deborah", "Ronald", "Stephanie",
	"Jason", "Rebecca", "Ryan", "Laura", "Jacob", "Sharon", "Gary", "Cynthia", "Nicholas", "Kathleen",
	"Eric", "Amy", "Jonathan", "Angela", "Justin", "Shirley", "Brandon", "Brenda", "Samuel", "Emma",
	"Carlos", "Maria", "Luis", "Sofia", "Miguel", "Camila", "Andre", "Aaliyah", "Wei", "Priya",
}

var LastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
	"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
	"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
	"Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell", "Carter", "Roberts",
	"Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker", "Cruz", "Edwards", "Collins", "Reyes",
	"Stewart", "Morris", "Morales", "Murphy", "Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper",
	"Peterson", "Bailey", "Reed", "Kelly", "Howard", "Kim", "Patel", "Chen", "Brooks", "Bennett",
}

var StreetNames = []string{
	"Main", "Oak", "Pine", "Maple", "Cedar", "Elm", "Washington", "Lake", "Hill", "Park",
	"Sunset", "Ridge", "Highland", "Willow", "Meadow", "River", "Forest", "Lincoln", "Jefferson", "Spring",
}

var StreetSuffixes = []string{"St", "Ave", "Blvd", "Dr", "Ln", "Rd", "Way", "Ct", "Pkwy"}

var EmailDomains = []string{"gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "icloud.com", "aol.com"}

var AgencyPrefixes = []string{
	"Summit", "Blue Ridge", "Northstar", "Pinnacle", "Evergreen", "Lakeside", "Ironwood", "Harbor",
	"Redwood", "Silver Oak", "Bright Path", "Keystone", "Granite Peak", "Copper Canyon", "Maple Leaf",
	"True North", "High Tide", "Clearwater", "Bold Arrow", "Golden Gate",
}

var AgencySuffixes = []string{
	"Marketing Group", "Growth Partners", "Digital Agency", "Media Co", "Lead Solutions",
	"Advertising", "Marketing Labs", "Digital Partners", "Growth Collective", "Sales Systems",
}

// BusinessTemplates are names of local businesses, with {last} for a last
// name and {city} for the city.
var BusinessTemplates = []string{
	"{last} Family Dental", "Bright Smile Orthodontics - {city}", "{city} Roofing & Solar", "{last} Insurance Agency",
	"{city} Home Remodeling", "{last} Law Group", "{city} Auto Care", "{last} Real Estate Team",
	"{city} Med Spa", "{last} Chiropractic", "{city} HVAC Pros", "{last} Plumbing & Drain",
	"{city} Fitness Studio", "{last} Mortgage Advisors", "{city} Pest Control", "{last} Landscaping",
	"{city} Window & Door", "{last} Financial Planning", "{city} Pool Services", "{last} Veterinary Clinic",
}

var SmallBusinessSuffixes = []string{"LLC", "& Sons", "Holdings", "Consulting", "Services", "Enterprises"}

var LeadSources = []string{
	"Facebook Ad", "Google Ads", "Website Form", "Referral", "Instagram", "Yelp", "Cold Call", "Walk In", "Direct Mail",
}

var LeadTags = []string{
	"hot lead", "warm lead", "facebook ad", "web form", "follow up", "referral", "appointment booked",
	"no answer", "callback requested", "vip", "spanish speaker", "reactivation",
}
