package adapter

import (
	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URL   string `default:"nats://localhost:4222"`
		Topic string `default:"leta"`
	}
	Index struct {
		Space string
	}
	Db struct {
		Driver     string
		Connection string
	}
	SQL struct {
		IndexSQLFile    string
		DocumentSQLFile string
	}
}

const prefix = "LRSQL"

// LoadConfig loads configuration variables from the environment
// and returns a fully populated Config instance.
func LoadConfig() (cfg Config, err error) {
	err = envconfig.CheckDisallowed(prefix, &cfg)
	if err != nil {
		return
	}

	err = envconfig.Process(prefix, &cfg)
	return
}

// Usage prints usage help to stdout
func Usage() {
	var cfg Config
	envconfig.Usage(prefix, &cfg)
}
