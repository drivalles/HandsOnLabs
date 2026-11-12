package ldspec

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// GHLConfig is the configuration of the GHL marketplace app.
type GHLConfig struct {
	// AppId is the id of the marketplace app.
	AppId string
	// SharedSecret is the key used to decrypt the user data GHL sends to the
	// app pages.
	SharedSecret string
	// OAuthCred are the OAuth credentials of the app.
	OAuthCred GHLClientCredentials
	// GHLBaseUrl is the base url of the GHL API, as GHL_OFFICIAL_API_BASE_URL
	// or a simulator.
	GHLBaseUrl string
	// ConversationProviderId is the conversation provider the dialer calls
	// are registered with.
	ConversationProviderId string
	Timeout                time.Duration
}

// GHLClientCredentials are the OAuth credentials of the GHL marketplace app.
type GHLClientCredentials struct {
	// ClientId is the OAuth client id of the app.
	ClientId string
	// ClientSecret is the OAuth client secret of the app.
	ClientSecret string
}

// GHLTestConfig is the GHL simulator of the tests.
type GHLTestConfig struct {
	// SimulatorIPAddr is the address the simulator listens on.
	SimulatorIPAddr string
	// SimulatorPort is the port the simulator listens on.
	SimulatorPort          int
	SimulatorSharedKey     string
	SimulatorJWTKey        string
	SimulatorAppId         string
	SimulatorDelay         time.Duration
	ClientCredentials      GHLClientCredentials
	ClientTimeout          time.Duration
	ConversationProviderId string
}

// GHLDataSize is the amount of data the GHL simulator generates.
type GHLDataSize int

const (
	// GHLDataSmall is one company with one location and a few records of
	// each kind.
	GHLDataSmall GHLDataSize = iota
	// GHLDataMedium is a few companies, each with some locations and
	// hundreds of records.
	GHLDataMedium
	// GHLDataLarge is at least 10 companies, each with many records and
	// hundreds of calls.
	GHLDataLarge
)

func (tCfg GHLTestConfig) ToGHLConfig() GHLConfig {
	return GHLConfig{
		AppId:                  tCfg.SimulatorAppId,
		SharedSecret:           tCfg.SimulatorSharedKey,
		OAuthCred:              tCfg.ClientCredentials,
		GHLBaseUrl:             fmt.Sprintf("http://%s:%d", tCfg.SimulatorIPAddr, tCfg.SimulatorPort),
		ConversationProviderId: tCfg.ConversationProviderId,
		Timeout:                tCfg.ClientTimeout,
	}
}

func (tCfg GHLTestConfig) SimulatorAddr() string {
	return net.JoinHostPort(tCfg.SimulatorIPAddr, strconv.Itoa(tCfg.SimulatorPort))
}
