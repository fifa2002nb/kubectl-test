package base

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
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
	configFlag := &genericclioptions.ConfigFlags{}
	namespace, _, err := configFlag.ToRawKubeConfigLoader().Namespace()
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	log.Infof("namespace:%s", namespace)
	clientConfig, err := configFlag.ToRESTConfig()
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(clientConfig)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	log.Infof("clientset:%v", clientset)
}
