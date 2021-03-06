package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/erkkah/letarette.sql/adapter"
	"github.com/erkkah/letarette/pkg/logger"
)

func main() {
	if len(os.Args) > 1 {
		adapter.Usage()
		os.Exit(0)
	}

	cfg, err := adapter.LoadConfig()
	if err != nil {
		logger.Error.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	logger.Info.Printf("letarette.sql %s (%s) starting, using %q driver", adapter.Revision, adapter.Tag, cfg.Db.Driver)

	errorHandler := func(err error) {
		logger.Error.Printf("Adapter error: %v", err)
	}
	a, err := adapter.New(cfg, errorHandler)
	if err != nil {
		logger.Error.Printf("Failed to initialize adapter: %v", err)
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	select {
	case <-signals:
		logger.Info.Printf("Closing down")
		a.Close()
	}
}
