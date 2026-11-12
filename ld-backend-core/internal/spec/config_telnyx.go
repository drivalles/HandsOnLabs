package ldspec

import "fmt"

type TelnyxConfig struct {
	BaseURL string
	ApiKey  string
}

type TelnyxTestConfig struct {
	IPAddr string
	Port   int
	ApiKey string
}

func (tCfg StripeTestConfig) ToTelnyxConfig() StripeConfig {
	return StripeConfig{
		BaseURL: fmt.Sprintf("http://%s:%d", tCfg.IPAddr, tCfg.Port),
		ApiKey:  tCfg.ApiKey,
	}
}
