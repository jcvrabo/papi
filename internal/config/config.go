package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	DBUrl            string `envconfig:"PAPI_DB_URL" required:"true"`
	OIDCIssuer       string `envconfig:"PAPI_OIDC_ISSUER" required:"true"`
	ListenAddr       string `envconfig:"PAPI_LISTEN_ADDR" default:":8080"`
	LogLevel         string `envconfig:"PAPI_LOG_LEVEL" default:"info"`
	MetricsAddr      string `envconfig:"PAPI_METRICS_ADDR" default:":9090"`
	IdentityCacheTTL int    `envconfig:"PAPI_IDENTITY_CACHE_TTL" default:"300"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
