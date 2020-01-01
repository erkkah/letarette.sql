package adapter

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the main configuration
type Config struct {
	Nats struct {
		URLS     []string `default:"nats://localhost:4222"`
		SeedFile string
		RootCAs  []string
		Topic    string `default:"leta"`
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

var usageFormat = fmt.Sprintf("{{$t:=\"\t\"}}letarette.sql\n%s (%s)\n", Tag, Revision) + `
Configuration environment variables:

VARIABLE{{$t}}TYPE{{$t}}DEFAULT
========{{$t}}===={{$t}}=======
LOG_LEVEL{{$t}}String{{$t}}INFO
{{range .}}{{if usage_description . | eq "internal" | not}}{{usage_key .}}{{$t}}{{usage_type .}}{{$t}}{{usage_default .}}
{{end}}{{end}}
`

// Usage prints usage help to stdout
func Usage() {
	var cfg Config
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	envconfig.Usagef(prefix, &cfg, tabs, usageFormat)
	fmt.Printf("Supported drivers: %s\n", strings.Join(getLoadedDrivers(), ", "))
}
