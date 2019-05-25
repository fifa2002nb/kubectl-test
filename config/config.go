package config

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/widuu/goini"
	"strconv"
)

// 配置项
type Options struct {
	// for common
	Port      int
	StoreFile string
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
		port := conf.GetValue("main", "port")
		if options.Port, err = strconv.Atoi(port); nil != err {
			log.Errorf("%v", err)
			options.Port = 8899
		}
		return options, nil
	} else {
		return nil, errors.New(fmt.Sprintf("configure is required to run a job. See '%s start --help'.", c.App.Name))
	}
}
