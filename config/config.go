package config

import (
	//"errors"
	//"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/widuu/goini"
	"strconv"
	"time"
)

// 配置项
type Options struct {
	// for agent
	Port                  int
	StreamIdleTimeout     time.Duration
	StreamCreationTimeout time.Duration

	// for cmd
	Namespace string
	PodName   string
	Image     string
	Command   string

	Agentless    bool
	AgentPodName string
}

// 解析配置文件
func ParseConf(c *cli.Context) (*Options, error) {
	options := &Options{}
	var err error
	var conf *goini.Config
	if c.IsSet("configure") || c.IsSet("C") {
		if c.IsSet("configure") {
			conf = goini.SetConfig(c.String("configure"))
		} else {
			conf = goini.SetConfig(c.String("C"))
		}

		// main configure
		port := conf.GetValue("agent", "port")
		if options.Port, err = strconv.Atoi(port); nil != err {
			log.Errorf("%v", err)
			options.Port = 8899
		}

		options.Namespace = conf.GetValue("cmd", "namespace")
		if "" == options.Namespace {
			options.Namespace = "hadoop"
		}

		options.PodName = conf.GetValue("cmd", "podname")
		if "" == options.PodName {
			options.PodName = "hdfs-namenode-0"
		}

		options.Image = conf.GetValue("cmd", "image")
		if "" == options.Image {
			options.Image = "nicolaka/netshoot:latest"
		}

		options.Command = conf.GetValue("cmd", "command")
		if "" == options.Command {
			options.Command = "bash"
		}
		options.StreamIdleTimeout = 10 * time.Minute
		options.StreamCreationTimeout = 15 * time.Second

		al := conf.GetValue("cmd", "agentless")
		if "true" == al {
			options.Agentless = true
		} else {
			options.Agentless = false
		}
		return options, nil
	} else {
		options.Port = 8899
		options.Namespace = "hadoop"
		options.PodName = "spark-base-0"
		options.Image = "nicolaka/netshoot:latest"
		options.Command = "bash"
		options.StreamIdleTimeout = 10 * time.Minute
		options.StreamCreationTimeout = 15 * time.Second
		options.Agentless = true
		//return nil, errors.New(fmt.Sprintf("configure is required to run a job. See '%s start --help'.", c.App.Name))
		return options, nil
	}
}
