package utils

import (
	"fmt"
	"ld-backend/internal/spec"
	"time"
)

var postgresIps = map[spec.TestNetwork]string{}
var cockroachIps = map[spec.TestNetwork]string{}
var ghlServerIps = map[spec.TestNetwork]string{}
var ldServerIps = map[spec.TestNetwork]string{}
var sipServerIps = map[spec.TestNetwork]string{}
var trunkServerIps = map[spec.TestNetwork]string{}

func init() {
	postgresIps[spec.TestNetworkInternal] = "127.90.8.1"
	postgresIps[spec.TestNetworkVlan] = "10.5.0.5"
	postgresIps[spec.TestNetworkLan] = "192.168.0.103"

	// the nodes of the cluster, the driver connects to the first one up
	cockroachIps[spec.TestNetworkInternal] = "127.90.9.1,127.90.9.2,127.90.9.3"
	cockroachIps[spec.TestNetworkVlan] = "10.5.1.5,10.5.1.6,10.5.1.7"
	cockroachIps[spec.TestNetworkLan] = "192.168.0.103"

	ghlServerIps[spec.TestNetworkInternal] = "127.0.22.1"
	ghlServerIps[spec.TestNetworkVlan] = "10.200.77.20"
	ghlServerIps[spec.TestNetworkLan] = "192.168.0.103"

	ldServerIps[spec.TestNetworkInternal] = "127.33.55.1"
	ldServerIps[spec.TestNetworkVlan] = "10.200.77.30"
	ldServerIps[spec.TestNetworkLan] = "192.168.0.103"

	sipServerIps[spec.TestNetworkInternal] = "127.33.55.1"
	sipServerIps[spec.TestNetworkVlan] = "10.200.77.30"
	sipServerIps[spec.TestNetworkLan] = "192.168.0.103"

	trunkServerIps[spec.TestNetworkInternal] = "127.44.10.1"
	trunkServerIps[spec.TestNetworkVlan] = "10.200.77.40"
	trunkServerIps[spec.TestNetworkLan] = "192.168.0.103"

}

type configService struct {
}

func NewConfigService() spec.ConfigService {
	return &configService{}
}

func (c configService) InMemoryDBConfig() spec.DBTestConfig {
	return spec.DBTestConfig{
		Driver:               spec.SqliteDriver,
		InMemorySQLite:       true,
		SQLiteDir:            "",
		CoreDbName:           "ldcore",
		TestTenantName:       "testtenant",
		PostgresTestIp:       "",
		PostgresTestPort:     0,
		PostgresTestUser:     "",
		PostgresTestPassword: "",
	}
}

func (c configService) TestdataDBFolderConfig() spec.DBTestConfig {
	return spec.DBTestConfig{
		Driver:               spec.SqliteDriver,
		InMemorySQLite:       false,
		SQLiteDir:            "testdata/dbs",
		CoreDbName:           "ldcore",
		TestTenantName:       "testtenant",
		PostgresTestIp:       "",
		PostgresTestPort:     0,
		PostgresTestUser:     "",
		PostgresTestPassword: "",
	}
}

func (c configService) PostgresDBConfig(net spec.TestNetwork) spec.DBTestConfig {
	return spec.DBTestConfig{
		Driver:               spec.PostgresDriver,
		InMemorySQLite:       false,
		SQLiteDir:            "",
		CoreDbName:           "ldcore",
		TestTenantName:       "testtenant",
		PostgresTestIp:       postgresIps[net],
		PostgresTestPort:     5432,
		PostgresTestUser:     "postgres",
		PostgresTestPassword: "0n89ynyYNny0dfy0n8989uy",
	}
}

func (c configService) CockroachDBConfig(net spec.TestNetwork) spec.DBTestConfig {
	return spec.DBTestConfig{
		Driver:               spec.CockroachDBDriver,
		InMemorySQLite:       false,
		SQLiteDir:            "",
		CoreDbName:           "ldcore",
		TestTenantName:       "testtenant",
		PostgresTestIp:       cockroachIps[net],
		PostgresTestPort:     26257,
		PostgresTestUser:     "leaddial",
		PostgresTestPassword: "90y78dfgsy34587ny9um8h980",
	}
}

func (c configService) GHLTestConfig() spec.GHLTestConfig {
	return c.GHLTestConfigWithNet(spec.TestNetworkInternal)
}

func (c configService) GHLTestConfigWithNet(net spec.TestNetwork) spec.GHLTestConfig {
	return spec.GHLTestConfig{
		SimulatorIPAddr:        ghlServerIps[net],
		SimulatorPort:          8686,
		SimulatorSharedKey:     "6b578567856785678n",
		SimulatorJWTKey:        GenerateRandomString(32),
		SimulatorAppId:         "app1",
		SimulatorDelay:         50 * time.Millisecond,
		ClientCredentials:      spec.GHLClientCredentials{},
		ClientTimeout:          2 * time.Second,
		ConversationProviderId: GenerateRandomString(12),
	}
}

func (c configService) InfoTestConfig() spec.InfoTestConfig {
	return c.InfoTestConfigWithNet(spec.TestNetworkInternal)
}

func (c configService) InfoTestConfigWithNet(net spec.TestNetwork) spec.InfoTestConfig {
	dbConfig := c.PostgresDBConfig(net)
	if net == spec.TestNetworkInternal {
		dbConfig = c.TestdataDBFolderConfig()
	}
	return spec.InfoTestConfig{
		GHLDataSize:   spec.GHLDataSmall,
		DBTestConfig:  dbConfig,
		GHLTestConfig: c.GHLTestConfigWithNet(net),
		NodeId:        "test-node",
		JWTSecret:     GenerateRandomString(32),
		RecordingsDir: "testdata/recordings",
		RecordingsUrl: fmt.Sprintf("https://%s:8443/recordings/%s", ldServerIps[net], "%s"),
	}
}

func (c configService) SipConfig(net spec.TestNetwork) spec.SIPConfig {
	return spec.SIPConfig{
		SIPAddr: spec.SIPServerAddr{
			Host: sipServerIps[net],
			Port: 5060,
		},
		Protocol: spec.UDP,
		MediaPorts: spec.PortRange{
			MinPort: 10000,
			MaxPort: 20000,
		},
		TrunkAddr: spec.SIPServerAddr{
			Host: trunkServerIps[net],
			Port: 5060,
		},
		TelnyxApiKey: "",
		DigestAuth:   spec.SIPAuth{},
		UAName:       "leaddial",
		Codecs:       spec.CODECS,
	}
}

func (c configService) SipSecureConfig(net spec.TestNetwork) spec.SIPConfig {
	return spec.SIPConfig{
		SIPAddr: spec.SIPServerAddr{
			Host: sipServerIps[net],
			Port: 5061,
		},
		Protocol: spec.TLS,
		MediaPorts: spec.PortRange{
			MinPort: 10000,
			MaxPort: 20000,
		},
		TLSCerts: spec.TLSCerts{
			CAFile:        "testdata/certs/ca.crt",
			CertFile:      "testdata/certs/certificate.crt",
			KeyFile:       "testdata/certs/certificate.key",
			FullChainFile: "testdata/certs/fullchain.crt",
		},
		TrunkAddr: spec.SIPServerAddr{
			Host: trunkServerIps[net],
			Port: 5061,
		},
		TelnyxApiKey: "",
		DigestAuth:   spec.SIPAuth{},
		UAName:       "leaddial",
		Codecs:       spec.CODECS,
	}
}

func (c configService) SipFsConfig() spec.SIPConfig {
	return spec.SIPConfig{
		SIPAddr: spec.SIPServerAddr{
			Host: sipServerIps[spec.TestNetworkInternal],
			Port: 5060,
		},
		Protocol: spec.UDP,
		MediaPorts: spec.PortRange{
			MinPort: 10000,
			MaxPort: 20000,
		},
		TrunkAddr: spec.SIPServerAddr{
			Host: "127.10.6.1",
			Port: 5060,
		},
		TLSCerts: spec.TLSCerts{
			CAFile:        "testdata/certs/ca.crt",
			CertFile:      "testdata/certs/certificate.crt",
			KeyFile:       "testdata/certs/certificate.key",
			FullChainFile: "testdata/certs/fullchain.crt",
		},
		TelnyxApiKey: "",
		DigestAuth: spec.SIPAuth{
			Username: "1004",
			Password: "1234",
		},
		UAName: "leaddial",
		Codecs: spec.CODECS,
	}
}

func (c configService) CallsEngineConfig(net spec.TestNetwork) spec.CallsEngineConfig {
	sipCfg := c.SipConfig(net)
	return spec.CallsEngineConfig{
		RecordingsDir: "testdata/recordings",
		IvrDir:        "testdata/ivr",
		IPAddr:        sipCfg.SIPAddr.Host,
		Hostname:      sipCfg.SIPAddr.Host,
		SIPConfig:     sipCfg,
	}
}

func (c configService) CallsEngineSecureConfig(net spec.TestNetwork) spec.CallsEngineConfig {
	sipCfg := c.SipSecureConfig(net)
	return spec.CallsEngineConfig{
		RecordingsDir: "testdata/recordings",
		IvrDir:        "testdata/ivr",
		IPAddr:        sipCfg.SIPAddr.Host,
		Hostname:      sipCfg.SIPAddr.Host,
		SIPConfig:     sipCfg,
	}
}

func (c configService) CallsEngineFsConfig() spec.CallsEngineConfig {
	sipCfg := c.SipFsConfig()
	return spec.CallsEngineConfig{
		RecordingsDir: "testdata/recordings",
		IvrDir:        "testdata/ivr",
		IPAddr:        sipCfg.SIPAddr.Host,
		Hostname:      sipCfg.SIPAddr.Host,
		SIPConfig:     sipCfg,
	}
}

func (c configService) LDAppTestConfig(net spec.TestNetwork) spec.LDAppTestConfig {
	ipAddr := ldServerIps[net]
	ghlCfg := c.GHLTestConfigWithNet(net)
	dbCfg := c.PostgresDBConfig(net)
	if net == spec.TestNetworkInternal {
		dbCfg = c.TestdataDBFolderConfig()
	}
	// the mock SIP trunk of the tests listens on the internal network
	sipCfg := c.SipSecureConfig(spec.TestNetworkInternal)
	sipCfg.SIPAddr.Host = ipAddr
	sipCfg.TLSCerts = spec.TLSCerts{}
	sipCfg.UAName = "1000"
	return spec.LDAppTestConfig{
		GHLTestConfig: ghlCfg,
		DBTestConfig:  dbCfg,
		AppConfig: spec.LDAppConfig{
			EnvName: "local",
			NodeId:  "e2e-test-node",
			WebConfig: spec.WebConfig{
				IPAddr:         ipAddr,
				Hostname:       ipAddr,
				BaseURL:        fmt.Sprintf("https://%s:8443", ipAddr),
				HTTPPort:       8443,
				JWTSecret:      GenerateRandomString(48),
				WebStaticDir:   "testdata/web/static",
				WebTemplateDir: "testdata/web/template",
				WebManagerDir:  "testdata/web/ld-manager",
			},
			CallsConfig: spec.CallsConfig{
				WebRTCConfig: spec.WebRTCConfig{
					IPAddr:      ipAddr,
					Codec:       spec.OPUS,
					WebRTCPorts: spec.PortRange{MinPort: 20001, MaxPort: 30000},
				},
				SIPConfig:     sipCfg,
				RecordingsDir: "testdata/recordings",
				IvrDir:        "testdata/ivr",
			},
			InfoConfig: spec.InfoConfig{
				DBConfig:  dbCfg.ToDBConfig(),
				GHLConfig: ghlCfg.ToGHLConfig(),
			},
			Certs: spec.TLSCerts{
				CAFile:        "testdata/certs/ca.crt",
				CertFile:      "testdata/certs/certificate.crt",
				KeyFile:       "testdata/certs/certificate.key",
				FullChainFile: "testdata/certs/certificate.crt",
			},
			MetricsPassword: "metrics",
		},
	}
}
