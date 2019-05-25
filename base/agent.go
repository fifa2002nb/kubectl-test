package base

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"kubectl-test/config"
	"kubectl-test/config/routes"
	"net/http"
	"os"
	"os/signal"
)

func Agent(c *cli.Context) {
	var (
		err     error
		options *config.Options
	)
	options, err = config.ParseConf(c)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", routes.Router())
		if err := http.ListenAndServe(fmt.Sprintf(":%d", options.Port), mux); nil != err {
			log.Fatal(err.Error())
			os.Exit(5)
		}
	}()
	waitingForExit()
}

func waitingForExit() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	killing := false
	for range sc {
		if killing {
			log.Info("Second interrupt: exiting")
			os.Exit(1)
		}
		killing = true
		go func() {
			log.Info("Interrupt: closing down...")
			log.Info("done")
			os.Exit(1)
		}()
	}
}
