package config

import (
	"errors"
	"fmt"
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
}

// 解析配置文件
func ParseConf(c *cli.Context) (*Options, error) {
	if c.IsSet("configure") || c.IsSet("C") {
		options := &Options{}
		var conf *goini.Config
		var err error
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
		return options, nil
	} else {
		return nil, errors.New(fmt.Sprintf("configure is required to run a job. See '%s start --help'.", c.App.Name))
	}
}
