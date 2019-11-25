package adapter

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URL   string `default:"nats://localhost:4222"`
		Topic string `default:"leta"`
	}
	Index struct {
		Space string `default:"docs"`
	}
	Db struct {
		Driver     string `required:"true"`
		Connection string
	}
	SQL struct {
		IndexSQLFile    string `default:"indexrequest.sql"`
		DocumentSQLFile string `default:"documentrequest.sql"`
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
	if err != nil {
		return
	}

	if !validateDriver(cfg.Db.Driver) {
		err = fmt.Errorf("No such driver: %q, supported drivers: %s",
			cfg.Db.Driver,
			strings.Join(getLoadedDrivers(), ", "),
		)
	}

	return
}

func validateDriver(driver string) bool {
	for _, loaded := range getLoadedDrivers() {
		if loaded == driver {
			return true
		}
	}
	return false
}

// Usage prints usage help to stdout
func Usage() {
	var cfg Config
	envconfig.Usage(prefix, &cfg)
}
