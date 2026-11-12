package ldspec

import "fmt"

type StripeConfig struct {
	BaseURL string
	ApiKey  string
}

type StripeTestConfig struct {
	IPAddr string
	Port   int
	ApiKey string
}

func (tCfg StripeTestConfig) ToStripeConfig() StripeConfig {
	return StripeConfig{
		BaseURL: fmt.Sprintf("http://%s:%d", tCfg.IPAddr, tCfg.Port),
		ApiKey:  tCfg.ApiKey,
	}
}
