package ldcfg

import (
	"fmt"
	ldutils "ld-backend/internal/impl/core/utils"
	"ld-backend/internal/spec"
	"time"
)

var postgresIps = map[ldspec.TestNetwork]string{}
var cockroachIps = map[ldspec.TestNetwork]string{}
var ghlServerIps = map[ldspec.TestNetwork]string{}
var ldServerIps = map[ldspec.TestNetwork]string{}
var sipServerIps = map[ldspec.TestNetwork]string{}
var trunkServerIps = map[ldspec.TestNetwork]string{}
var telnyxServerIps = map[ldspec.TestNetwork]string{}
var stripeServerIps = map[ldspec.TestNetwork]string{}
var kafkaServerIps = map[ldspec.TestNetwork]string{}

var lanIp = "192.168.0.103"

func init() {
	postgresIps[ldspec.TestNetworkInternal] = "127.90.8.1"
	postgresIps[ldspec.TestNetworkVlan] = "10.5.0.5"

	// the nodes of the cluster, the driver connects to the first one up
	cockroachIps[ldspec.TestNetworkInternal] = "127.90.9.1,127.90.9.2,127.90.9.3"
	cockroachIps[ldspec.TestNetworkVlan] = "10.5.1.5,10.5.1.6,10.5.1.7"

	ghlServerIps[ldspec.TestNetworkInternal] = "127.90.22.1"
	ghlServerIps[ldspec.TestNetworkVlan] = "10.200.77.20"

	ldServerIps[ldspec.TestNetworkInternal] = "127.90.55.1"
	ldServerIps[ldspec.TestNetworkVlan] = "10.200.77.30"

	sipServerIps[ldspec.TestNetworkInternal] = "127.90.55.1"
	sipServerIps[ldspec.TestNetworkVlan] = "10.200.77.30"

	trunkServerIps[ldspec.TestNetworkInternal] = "127.90.10.1"
	trunkServerIps[ldspec.TestNetworkVlan] = "10.200.77.40"

	telnyxServerIps[ldspec.TestNetworkInternal] = "127.90.71.1"
	telnyxServerIps[ldspec.TestNetworkVlan] = "10.200.71.40"

	stripeServerIps[ldspec.TestNetworkInternal] = "127.90.61.1"
	stripeServerIps[ldspec.TestNetworkVlan] = "10.200.61.40"

	kafkaServerIps[ldspec.TestNetworkInternal] = "127.90.47.1"
	kafkaServerIps[ldspec.TestNetworkVlan] = "10.5.2.1"

}

func getIp(net ldspec.TestNetwork, ips map[ldspec.TestNetwork]string) string {
	if net == ldspec.TestNetworkLan {
		return lanIp
	}
	return ips[net]
}

type configService struct {
}

func (c configService) EventsTestConfig(net ldspec.TestNetwork) ldspec.EventsTestConfig {
	return ldspec.EventsTestConfig{
		KafkaIpAddr: kafkaServerIps[net],
	}
}

func NewConfigService() ldspec.TestConfigService {
	return &configService{}
}

func (c configService) InMemoryDBConfig() ldspec.DBTestConfig {
	return ldspec.DBTestConfig{
		Driver:             ldspec.SqliteDriver,
		InMemorySQLite:     true,
		SQLiteDir:          "",
		CoreDbName:         "ldcore",
		TestTenantName:     "testtenant",
		TestServerIp:       "",
		TestServerPort:     0,
		TestServerUser:     "",
		TestServerPassword: "",
	}
}

func (c configService) TestdataDBFolderConfig() ldspec.DBTestConfig {
	return ldspec.DBTestConfig{
		Driver:             ldspec.SqliteDriver,
		InMemorySQLite:     false,
		SQLiteDir:          "testdata/dbs",
		CoreDbName:         "ldcore",
		TestTenantName:     "testtenant",
		TestServerIp:       "",
		TestServerPort:     0,
		TestServerUser:     "",
		TestServerPassword: "",
	}
}

func (c configService) PostgresDBConfig(net ldspec.TestNetwork) ldspec.DBTestConfig {
	return ldspec.DBTestConfig{
		Driver:             ldspec.PostgresDriver,
		InMemorySQLite:     false,
		SQLiteDir:          "",
		CoreDbName:         "ldcore",
		TestTenantName:     "testtenant",
		TestServerIp:       getIp(net, postgresIps),
		TestServerPort:     5432,
		TestServerUser:     "postgres",
		TestServerPassword: "0n89ynyYNny0dfy0n8989uy",
	}
}

func (c configService) CockroachDBConfig(net ldspec.TestNetwork) ldspec.DBTestConfig {
	return ldspec.DBTestConfig{
		Driver:             ldspec.CockroachDBDriver,
		InMemorySQLite:     false,
		SQLiteDir:          "",
		CoreDbName:         "ldcore",
		TestTenantName:     "testtenant",
		TestServerIp:       getIp(net, cockroachIps),
		TestServerPort:     26257,
		TestServerUser:     "leaddial",
		TestServerPassword: "90y78dfgsy34587ny9um8h980",
	}
}

func (c configService) GHLTestConfig() ldspec.GHLTestConfig {
	return c.GHLTestConfigWithNet(ldspec.TestNetworkInternal)
}

func (c configService) GHLTestConfigWithNet(net ldspec.TestNetwork) ldspec.GHLTestConfig {
	return ldspec.GHLTestConfig{
		SimulatorIPAddr:        getIp(net, ghlServerIps),
		SimulatorPort:          8686,
		SimulatorSharedKey:     "6b578567856785678n",
		SimulatorJWTKey:        ldutils.GenerateRandomString(32),
		SimulatorAppId:         "app1",
		SimulatorDelay:         50 * time.Millisecond,
		ClientCredentials:      ldspec.GHLClientCredentials{},
		ClientTimeout:          2 * time.Second,
		ConversationProviderId: ldutils.GenerateRandomString(12),
	}
}

func (c configService) InfoTestConfig() ldspec.InfoTestConfig {
	return c.InfoTestConfigWithNet(ldspec.TestNetworkInternal)
}

func (c configService) InfoTestConfigWithNet(net ldspec.TestNetwork) ldspec.InfoTestConfig {
	dbConfig := c.PostgresDBConfig(net)
	if net == ldspec.TestNetworkInternal {
		dbConfig = c.TestdataDBFolderConfig()
	}
	return ldspec.InfoTestConfig{
		GHLDataSize:   ldspec.GHLDataSmall,
		DBTestConfig:  dbConfig,
		GHLTestConfig: c.GHLTestConfigWithNet(net),
		NodeId:        "test-node",
		JWTSecret:     ldutils.GenerateRandomString(32),
		RecordingsDir: "testdata/recordings",
		RecordingsUrl: fmt.Sprintf("https://%s:8443/recordings/%s", getIp(net, ldServerIps), "%s"),
	}
}

func (c configService) SipConfig(net ldspec.TestNetwork) ldspec.SIPConfig {
	return ldspec.SIPConfig{
		SIPAddr: ldspec.SIPServerAddr{
			Host: getIp(net, sipServerIps),
			Port: 5060,
		},
		Protocol: ldspec.UDP,
		MediaPorts: ldspec.PortRange{
			MinPort: 10000,
			MaxPort: 20000,
		},
		TrunkAddr: ldspec.SIPServerAddr{
			Host: getIp(net, trunkServerIps),
			Port: 5060,
		},
		TelnyxApiKey: "",
		DigestAuth:   ldspec.SIPAuth{},
		UAName:       "leaddial",
		Codecs:       ldspec.CODECS,
	}
}

func (c configService) SipSecureConfig(net ldspec.TestNetwork) ldspec.SIPConfig {
	return ldspec.SIPConfig{
		SIPAddr: ldspec.SIPServerAddr{
			Host: getIp(net, sipServerIps),
			Port: 5061,
		},
		Protocol: ldspec.TLS,
		MediaPorts: ldspec.PortRange{
			MinPort: 10000,
			MaxPort: 20000,
		},
		TLSCerts: ldspec.TLSCerts{
			CAFile:        "testdata/certs/ca.crt",
			CertFile:      "testdata/certs/certificate.crt",
			KeyFile:       "testdata/certs/certificate.key",
			FullChainFile: "testdata/certs/fullchain.crt",
		},
		TrunkAddr: ldspec.SIPServerAddr{
			Host: getIp(net, trunkServerIps),
			Port: 5061,
		},
		TelnyxApiKey: "",
		DigestAuth:   ldspec.SIPAuth{},
		UAName:       "leaddial",
		Codecs:       ldspec.CODECS,
	}
}

func (c configService) SipFsConfig() ldspec.SIPConfig {
	return ldspec.SIPConfig{
		SIPAddr: ldspec.SIPServerAddr{
			Host: getIp(ldspec.TestNetworkInternal, sipServerIps),
			Port: 5060,
		},
		Protocol: ldspec.UDP,
		MediaPorts: ldspec.PortRange{
			MinPort: 10000,
			MaxPort: 20000,
		},
		TrunkAddr: ldspec.SIPServerAddr{
			Host: "127.10.6.1",
			Port: 5060,
		},
		TLSCerts: ldspec.TLSCerts{
			CAFile:        "testdata/certs/ca.crt",
			CertFile:      "testdata/certs/certificate.crt",
			KeyFile:       "testdata/certs/certificate.key",
			FullChainFile: "testdata/certs/fullchain.crt",
		},
		TelnyxApiKey: "",
		DigestAuth: ldspec.SIPAuth{
			Username: "1004",
			Password: "1234",
		},
		UAName: "leaddial",
		Codecs: ldspec.CODECS,
	}
}

func (c configService) CallsEngineConfig(net ldspec.TestNetwork) ldspec.CallsEngineConfig {
	sipCfg := c.SipConfig(net)
	return ldspec.CallsEngineConfig{
		RecordingsDir: "testdata/recordings",
		IvrDir:        "testdata/ivr",
		IPAddr:        sipCfg.SIPAddr.Host,
		Hostname:      sipCfg.SIPAddr.Host,
		SIPConfig:     sipCfg,
	}
}

func (c configService) CallsEngineSecureConfig(net ldspec.TestNetwork) ldspec.CallsEngineConfig {
	sipCfg := c.SipSecureConfig(net)
	return ldspec.CallsEngineConfig{
		RecordingsDir: "testdata/recordings",
		IvrDir:        "testdata/ivr",
		IPAddr:        sipCfg.SIPAddr.Host,
		Hostname:      sipCfg.SIPAddr.Host,
		SIPConfig:     sipCfg,
	}
}

func (c configService) CallsEngineFsConfig() ldspec.CallsEngineConfig {
	sipCfg := c.SipFsConfig()
	return ldspec.CallsEngineConfig{
		RecordingsDir: "testdata/recordings",
		IvrDir:        "testdata/ivr",
		IPAddr:        sipCfg.SIPAddr.Host,
		Hostname:      sipCfg.SIPAddr.Host,
		SIPConfig:     sipCfg,
	}
}

func (c configService) LDAppTestConfig(net ldspec.TestNetwork) ldspec.LDAppTestConfig {
	ipAddr := getIp(net, ldServerIps)
	ghlCfg := c.GHLTestConfigWithNet(net)
	dbCfg := c.PostgresDBConfig(net)
	if net == ldspec.TestNetworkInternal {
		dbCfg = c.TestdataDBFolderConfig()
	}
	// the mock SIP trunk of the tests listens on the internal network
	sipCfg := c.SipSecureConfig(ldspec.TestNetworkInternal)
	sipCfg.SIPAddr.Host = ipAddr
	sipCfg.TLSCerts = ldspec.TLSCerts{}
	sipCfg.UAName = "1000"
	return ldspec.LDAppTestConfig{
		GHLTestConfig: ghlCfg,
		DBTestConfig:  dbCfg,
		AppConfig: ldspec.LDAppConfig{
			EnvName: "local",
			NodeId:  "e2e-test-node",
			WebConfig: ldspec.WebConfig{
				IPAddr:         ipAddr,
				Hostname:       ipAddr,
				BaseURL:        fmt.Sprintf("https://%s:8443", ipAddr),
				HTTPPort:       8443,
				JWTSecret:      ldutils.GenerateRandomString(48),
				WebStaticDir:   "testdata/web/static",
				WebTemplateDir: "testdata/web/template",
				WebManagerDir:  "testdata/web/ld-manager",
			},
			CallsConfig: ldspec.CallsConfig{
				WebRTCConfig: ldspec.WebRTCConfig{
					IPAddr:      ipAddr,
					Codec:       ldspec.OPUS,
					WebRTCPorts: ldspec.PortRange{MinPort: 20001, MaxPort: 30000},
				},
				SIPConfig:     sipCfg,
				RecordingsDir: "testdata/recordings",
				IvrDir:        "testdata/ivr",
			},
			InfoConfig: ldspec.InfoConfig{
				DBConfig:  dbCfg.ToDBConfig(),
				GHLConfig: ghlCfg.ToGHLConfig(),
			},
			Certs: ldspec.TLSCerts{
				CAFile:        "testdata/certs/ca.crt",
				CertFile:      "testdata/certs/certificate.crt",
				KeyFile:       "testdata/certs/certificate.key",
				FullChainFile: "testdata/certs/certificate.crt",
			},
			MetricsPassword: "metrics",
		},
	}
}

func (c configService) TelnyxTestConfig(net ldspec.TestNetwork) ldspec.TelnyxTestConfig {
	return ldspec.TelnyxTestConfig{
		IPAddr: getIp(net, telnyxServerIps),
		Port:   7766,
		ApiKey: "u890fg0y89ny708ndfy87dfgy87n0",
	}
}

func (c configService) StripeTestConfig(net ldspec.TestNetwork) ldspec.StripeTestConfig {
	return ldspec.StripeTestConfig{
		IPAddr: getIp(net, telnyxServerIps),
		Port:   9988,
		ApiKey: "98ut45n8y0t6723456ee6w4546x3wwes",
	}
}
