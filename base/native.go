package base

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"kubectl-test/config"
	"kubectl-test/utils/runtime"
	"os"
)

func Native(c *cli.Context) {
	var (
		err     error
		options *config.Options
	)
	options, err = config.ParseConf(c)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	} else {
		log.Infof("options:%v", options)
	}
	log.Infof("namespace:%v, podname:%v, image:%v, command:%v", options.Namespace, options.PodName, options.Image, options.Command)
	client, err := runtime.NewKubeDockerClient()
	if nil != err {
		log.Fatalf("%v", err)
		os.Exit(1)
	}
	cxt, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = client.PullImage(options.Image, nil, cxt)
	if nil != err {
		log.Fatalf("%v", err)
		os.Exit(1)
	}
}
