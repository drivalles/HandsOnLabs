package ldspec

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
