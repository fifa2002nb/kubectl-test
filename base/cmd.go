package base

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"kubectl-test/config"
	"os"
)

func Cmd(c *cli.Context) {
	var (
		err     error
		options *config.Options
	)
	options, err = config.ParseConf(c)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	} else {
		log.Infof("%v", options)
	}
}
