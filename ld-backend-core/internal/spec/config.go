package spec

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// LDAppConfig is the configuration of the LeadDial server: the web server,
// the calls and the data of the GHL locations.
type LDAppConfig struct {
	// EnvName is the name of the environment, as local or prod; it selects
	// the color and the title of the user interface.
	EnvName string
	// NodeId identifies the server node; the locks and the dialer sessions
	// of a node are released when it starts again.
	NodeId string
	// WebConfig is the configuration of the web server.
	WebConfig WebConfig
	// CallsConfig is the configuration of the WebRTC and SIP calls.
	CallsConfig CallsConfig
	// InfoConfig is the configuration of the databases and of GHL.
	InfoConfig InfoConfig
	// Certs are the TLS certificates of the web and SIP servers.
	Certs TLSCerts
	// MetricsPassword is the password of the ldserver-metrics-parser user,
	// which reads the metrics of the server.
	MetricsPassword string
}

// LDAppTestConfig is the configuration of the server for the tests, with
// the GHL simulator and the databases they run with.
type LDAppTestConfig struct {
	// AppConfig is the configuration of the server under test.
	AppConfig LDAppConfig
	// GHLTestConfig is the GHL simulator the server calls.
	GHLTestConfig GHLTestConfig
	// DBTestConfig are the databases of the tests.
	DBTestConfig DBTestConfig
}

// WebConfig is the configuration of the HTTPS server of the app pages, the
// API and the dialer signals.
type WebConfig struct {
	// IPAddr is the address the server listens on.
	IPAddr string
	// Hostname is the public name of the server, also used as the hostname
	// of the SIP user agent.
	Hostname string
	// BaseURL is the public url of the server, as https://host:port; the
	// recordings are served under it.
	BaseURL string
	// HTTPPort is the port of the HTTPS server.
	HTTPPort int
	// JWTSecret signs the tokens of the dialer sessions of the users.
	JWTSecret string
	// WebManagerDir is the folder of the manager web app.
	WebManagerDir string
	// WebStaticDir is the folder of the static files, served under /static.
	WebStaticDir string
	// WebTemplateDir is the folder of the page templates, the *.tpl files.
	WebTemplateDir string
}

// CallsConfig is the configuration of the calls of the dialer.
type CallsConfig struct {
	// WebRTCConfig is the audio of the users in the browser.
	WebRTCConfig WebRTCConfig
	// SIPConfig is the calls to the contacts through the SIP trunk.
	SIPConfig SIPConfig
	// RecordingsDir is the folder where the calls are recorded.
	RecordingsDir string
	// IvrDir is the folder of the audio files played in the calls, as the
	// voicemail drop.
	IvrDir string
}

// InfoConfig is the configuration of the data of the GHL locations.
type InfoConfig struct {
	// DBConfig are the core database and the database of each location.
	DBConfig DBConfig
	// GHLConfig is the GHL marketplace app the server is.
	GHLConfig     GHLConfig
	NodeId        string
	JWTSecret     string
	RecordingsDir string
	RecordingsUrl string
}

type InfoTestConfig struct {
	GHLDataSize GHLDataSize
	// DBConfig are the core database and the database of each location.
	DBTestConfig DBTestConfig
	// GHLConfig is the GHL marketplace app the server is.
	GHLTestConfig GHLTestConfig
	// RecordingsUrl is the url template of a recording, with a %s for its
	// file; GHL gets the recordings of the calls from it.
	NodeId        string
	JWTSecret     string
	RecordingsDir string
	RecordingsUrl string
}

func (cfg InfoTestConfig) ToInfoConfig() InfoConfig {
	return InfoConfig{
		DBConfig:      cfg.DBTestConfig.ToDBConfig(),
		GHLConfig:     cfg.GHLTestConfig.ToGHLConfig(),
		NodeId:        cfg.NodeId,
		JWTSecret:     cfg.JWTSecret,
		RecordingsDir: cfg.RecordingsDir,
		RecordingsUrl: cfg.RecordingsUrl,
	}
}

// TLSCerts are the files of the TLS certificates of the server.
type TLSCerts struct {
	// CertFile is the certificate of the server.
	CertFile string
	// KeyFile is the private key of the certificate.
	KeyFile string
	// CAFile is the certificate authority the SIP peers are verified with;
	// empty uses the certificates of the system.
	CAFile string
	// FullChainFile is the certificate with its chain, served by the HTTPS
	// server.
	FullChainFile string
}

// WebRTCConfig is the configuration of the WebRTC audio of the users.
type WebRTCConfig struct {
	// WebRTCPorts are the UDP ports of the WebRTC media; the minimum must be
	// at least 1000.
	WebRTCPorts PortRange
	// IPAddr is the address of the WebRTC media.
	IPAddr string
	// Codec is the audio codec of the WebRTC calls.
	Codec RTCCodec
}

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

// PortRange is a range of ports, both ends included.
type PortRange struct {
	MinPort uint16
	MaxPort uint16
}

// RTCCodec is an audio codec of the WebRTC and SIP calls.
type RTCCodec string

const (
	// NONE is no codec selected.
	NONE RTCCodec = ""
	// PCMU is G.711 µ-law, 8 kHz.
	PCMU RTCCodec = "pcmu"
	// PCMA is G.711 A-law, 8 kHz.
	PCMA RTCCodec = "pcma"
	// G722 is G.722 wideband, 16 kHz.
	G722 RTCCodec = "g722"
	// OPUS is Opus, the codec of the WebRTC calls.
	OPUS RTCCodec = "opus"
)

// CODECS are all the supported codecs, by preference.
var CODECS = []RTCCodec{OPUS, G722, PCMU, PCMA}

// SIPConfig is the configuration of the SIP calls to the contacts.
type SIPConfig struct {
	// SIPAddr is the address and port the SIP server listens on.
	SIPAddr SIPServerAddr
	// Protocol is the SIP transport, as TLS.
	Protocol SIPProtocol
	// MediaPorts are the UDP ports of the RTP media of the calls.
	MediaPorts PortRange
	TLSCerts   TLSCerts
	// TrunkAddr is the SIP trunk the calls are sent to.
	TrunkAddr SIPServerAddr
	// TelnyxApiKey is the key of the Telnyx API, when the trunk is Telnyx.
	TelnyxApiKey string
	// DigestAuth are the credentials of the trunk; empty when the trunk does
	// not ask for them.
	DigestAuth SIPAuth
	// UAName is the default name of the SIP user agent.
	UAName string
	// Codecs are the codecs offered to the trunk, by preference.
	Codecs []RTCCodec
}

type CallsEngineConfig struct {
	IPAddr        string
	Hostname      string
	SIPConfig     SIPConfig
	RecordingsDir string
	IvrDir        string
}

// SIPServerAddr is the address of a SIP server.
type SIPServerAddr struct {
	Host string
	Port int
}

// SIPProtocol is the transport of the SIP messages.
type SIPProtocol string

const (
	UDP SIPProtocol = "UDP"
	TCP SIPProtocol = "TCP"
	TLS SIPProtocol = "TLS"
)

// SIPAuth are the credentials of the SIP digest authentication.
type SIPAuth struct {
	Username string
	Password string
}

// DBConfig is the configuration to connect to the core and tenant databases.
type DBConfig struct {
	// Driver is the database engine used for the core and tenant databases.
	Driver LDDatabaseDriver
	// CoreDBName is the name of the core database, put into DbUrlTpl.
	CoreDBName string
	// DbUrlTpl is the connection url with a %s placeholder for the database
	// name; it is used for the core database and for each tenant database.
	DbUrlTpl string
}

// LDDatabaseDriver is the database engine used by the repositories.
type LDDatabaseDriver string

const (
	// PostgresDriver connects to PostgreSQL; tenant databases are created on
	// the server when missing.
	PostgresDriver LDDatabaseDriver = "postgres"
	// SqliteDriver connects to SQLite files, used for tests and local runs.
	SqliteDriver LDDatabaseDriver = "sqlite"
	// CockroachDBDriver connects to CockroachDB with the postgres driver.
	CockroachDBDriver LDDatabaseDriver = "cockroachdb"
)

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

// DBTestConfig are the databases of the tests.
type DBTestConfig struct {
	Driver LDDatabaseDriver
	// InMemorySQLite keeps the SQLite databases in memory, without files.
	InMemorySQLite bool
	// SQLiteDir is the folder of the SQLite database files.
	SQLiteDir      string
	CoreDbName     string
	TestTenantName string
	// PostgresTestIp is the address of the server, or the addresses of the
	// nodes of a cluster separated by commas.
	PostgresTestIp       string
	PostgresTestPort     int
	PostgresTestUser     string
	PostgresTestPassword string
}

type TestNetwork string

const (
	TestNetworkInternal TestNetwork = "internal"
	TestNetworkVlan     TestNetwork = "vlan"
	TestNetworkLan      TestNetwork = "lan"
)

type ConfigService interface {
	InMemoryDBConfig() DBTestConfig
	TestdataDBFolderConfig() DBTestConfig
	PostgresDBConfig(net TestNetwork) DBTestConfig
	CockroachDBConfig(net TestNetwork) DBTestConfig
	GHLTestConfig() GHLTestConfig
	GHLTestConfigWithNet(net TestNetwork) GHLTestConfig
	InfoTestConfig() InfoTestConfig
	InfoTestConfigWithNet(net TestNetwork) InfoTestConfig
	SipConfig(net TestNetwork) SIPConfig
	SipSecureConfig(net TestNetwork) SIPConfig
	SipFsConfig() SIPConfig
	CallsEngineConfig(net TestNetwork) CallsEngineConfig
	CallsEngineSecureConfig(net TestNetwork) CallsEngineConfig
	CallsEngineFsConfig() CallsEngineConfig
	// LDAppTestConfig is the configuration of the whole server on the test
	// network, with the GHL simulator and the databases it runs with.
	LDAppTestConfig(net TestNetwork) LDAppTestConfig
}

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

func (tCfg DBTestConfig) ToDBConfig() DBConfig {
	cfg := DBConfig{
		Driver:     tCfg.Driver,
		CoreDBName: tCfg.CoreDbName,
	}
	if tCfg.Driver == SqliteDriver {
		if tCfg.InMemorySQLite {
			cfg.DbUrlTpl = "file:%s?mode=memory&cache=shared&loc=auto"
		} else {
			cfg.DbUrlTpl = fmt.Sprintf("%s/%s.db?_journal_mode=WAL&_busy_timeout=5000&_txlock=immediate", tCfg.SQLiteDir, "%s")
		}
	} else if tCfg.Driver == PostgresDriver {
		cfg.DbUrlTpl = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			tCfg.PostgresTestIp, tCfg.PostgresTestPort, tCfg.PostgresTestUser, tCfg.PostgresTestPassword, "%s")
	} else if tCfg.Driver == CockroachDBDriver {
		// the test server runs in secure mode, so it only takes TLS
		cfg.DbUrlTpl = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
			tCfg.PostgresTestIp, tCfg.PostgresTestPort, tCfg.PostgresTestUser, tCfg.PostgresTestPassword, "%s")
	} else {
		panic(fmt.Sprintf("unknown driver %s", tCfg.Driver))
	}
	return cfg
}
