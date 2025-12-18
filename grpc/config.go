package grpc

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ShutdownTimeout time.Duration `default:"20s"`
	APIHost         string        `default:"0.0.0.0:50051"`
	DebugHost       string        `default:"0.0.0.0:3010"`
	MetricsHost     string        `default:"0.0.0.0:2112"`
	Build           string        `default:"dev"`
	Desc            string        `default:"example grpc server"`
	Namespace       string        `default:"test"`
	Version         string        `default:"test"`
	Name            string        `default:"test"`
}

func LoadConfig(prefix string) (Config, error) {
	var c Config
	err := envconfig.Process(prefix, &c)
	if err != nil {
		return c, err
	}

	if c.Namespace == "" {
		c.Namespace = prefix
	}

	return c, nil
}
