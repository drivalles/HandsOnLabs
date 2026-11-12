package ldspec

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

// PortRange is a range of ports, both ends included.
type PortRange struct {
	MinPort uint16
	MaxPort uint16
}

type TestNetwork string

const (
	TestNetworkInternal TestNetwork = "internal"
	TestNetworkVlan     TestNetwork = "vlan"
	TestNetworkLan      TestNetwork = "lan"
)

type TestConfigService interface {
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
	TelnyxTestConfig(net TestNetwork) TelnyxTestConfig
	StripeTestConfig(net TestNetwork) StripeTestConfig
	EventsTestConfig(net TestNetwork) EventsTestConfig
	// LDAppTestConfig is the configuration of the whole server on the test
	// network, with the GHL simulator and the databases it runs with.
	LDAppTestConfig(net TestNetwork) LDAppTestConfig
}
