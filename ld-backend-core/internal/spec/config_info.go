package ldspec

// InfoConfig is the configuration of the info service, the business layer of
// the GHL locations.
type InfoConfig struct {
	// DBConfig are the core database and the database of each location.
	DBConfig DBConfig
	// GHLConfig is the GHL marketplace app the server is.
	GHLConfig GHLConfig
	// NodeId is the id of the server node, which holds the locks of the
	// contacts and DIDs its dialers call.
	NodeId string
	// JWTSecret is the secret used to sign the JWT tokens for all user sessions.
	JWTSecret string
	// RecordingsDir is the folder of the recordings of the calls.
	RecordingsDir string
	// RecordingsUrl is the url template of a recording, with a %s for its
	// file; GHL gets the recordings of the calls from it.
	RecordingsUrl string
}

// InfoTestConfig is the configuration of the info service in the tests, with
// the test databases and the GHL simulator.
type InfoTestConfig struct {
	// GHLDataSize is the size of the data the GHL simulator generates.
	GHLDataSize GHLDataSize
	// DBConfig are the core database and the database of each location.
	DBTestConfig DBTestConfig
	// GHLConfig is the GHL marketplace app the server is.
	GHLTestConfig GHLTestConfig
	// NodeId is the id of the server node.
	NodeId string
	// JWTSecret is the secret used to sign the JWT tokens for all user sessions.
	JWTSecret string
	// RecordingsDir is the folder of the recordings of the calls.
	RecordingsDir string
	// RecordingsUrl is the url template of a recording, with a %s for its
	// file; GHL gets the recordings of the calls from it.
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
