package rest

import (
	"time"
)

type Config struct {
	ReadTimeout        time.Duration `default:"5s"`
	WriteTimeout       time.Duration `default:"10s"`
	IdleTimeout        time.Duration `default:"120s"`
	ShutdownTimeout    time.Duration `default:"20s"`
	APIHost            string        `default:"0.0.0.0:3000"`
	DebugHost          string        `default:"0.0.0.0:3010"`
	MetricsHost        string        `default:"0.0.0.0:2112"`
	CorsAllowedOrigins []string      `default:"*"`
	MaxHeaderBytes     int           `default:"0"`
	Build              string        `default:"dev"`
	Desc               string        `default:"example server"`
	Namespace          string
}
