package ldspec

type EventsConfig struct {
	KafkaIpAddr string
}

type EventsTestConfig struct {
	KafkaIpAddr string
}

func (e EventsTestConfig) ToEventsConfig() EventsConfig {
	return EventsConfig{
		KafkaIpAddr: e.KafkaIpAddr,
	}
}
