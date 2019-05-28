package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	base "kubectl-test/base"
	"kubectl-test/version"
	"os"
	"path"
	"runtime"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   false,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000000",
	})
	runtime.GOMAXPROCS(40)
}

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "kubectl plugin tool"
	app.Version = version.VERSION + " (" + version.GITCOMMIT + ")"
	app.Author = ""
	app.Email = ""
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug mode",
			EnvVar: "DEBUG",
		},

		cli.StringFlag{
			Name:  "log-level, l",
			Value: "info",
			Usage: fmt.Sprintf("Log level (options: debug, info, warn, error, fatal, panic)"),
		},
	}

	// logs
	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.SetLevel(level)

		// If a log level wasn't specified and we are running in debug mode,
		// enforce log-level=debug.
		if !c.IsSet("log-level") && !c.IsSet("l") && c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:      "agent",
			ShortName: "a",
			Usage:     "agent",
			Flags:     []cli.Flag{base.FlConf},
			Action:    base.Agent,
		},
		{
			Name:      "cmd",
			ShortName: "c",
			Usage:     "cmd",
			Flags:     []cli.Flag{base.FlConf},
			Action:    base.Cmd,
		},
		{
			Name:      "native",
			ShortName: "n",
			Usage:     "native",
			Flags:     []cli.Flag{base.FlConf},
			Action:    base.Native,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
