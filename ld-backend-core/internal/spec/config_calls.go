package ldspec

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
