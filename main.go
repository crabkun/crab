package main

import (
	"encoding/json"
	"flag"
	"github.com/crabkun/crab/config"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

const VERSION = "beta2"

var (
	ConfigFile string
)

func init() {
	flag.StringVar(&ConfigFile, "config", "config.json", "config file")
}

func main() {
	flag.Parse()
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
	})

	var err error
	log.Infoln("Crab", VERSION)

	l := log.WithFields(log.Fields{
		"file": ConfigFile,
	})

	cfgBuf, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		l.WithError(err).Fatalln("load configure file failed")
	}

	var baseCfg *config.BaseConfig
	err = json.Unmarshal(cfgBuf, &baseCfg)
	if err != nil {
		l.WithError(err).Fatalln("unmarshal configure file failed")
	}
	err = baseCfg.Validate()
	if err != nil {
		l.WithError(err).Fatalln("validate configure file failed")
	}

	logLvl, err := log.ParseLevel(baseCfg.LogLevel)
	if err != nil {
		l.WithError(err).WithFields(log.Fields{
			"log_level": baseCfg.LogLevel,
		}).Fatalln("unsupported log level")
	}
	log.SetLevel(logLvl)

	switch strings.ToLower(baseCfg.Mode) {
	case "client":
		var clientConfig *config.ClientConfig
		err = json.Unmarshal(cfgBuf, &clientConfig)
		if err != nil {
			l.WithError(err).Fatalln("unmarshal client configure file failed")
		}
		err = clientConfig.Validate()
		if err != nil {
			l.WithError(err).Fatalln("validate client configure file failed")
		}

		ClientMain(clientConfig)
	case "server":
		var serverConfig *config.ServerConfig
		err = json.Unmarshal(cfgBuf, &serverConfig)
		if err != nil {
			l.WithError(err).Fatalln("unmarshal server configure file failed")
		}
		err = serverConfig.Validate()
		if err != nil {
			l.WithError(err).Fatalln("validate server configure file failed")
		}

		ServerMain(serverConfig)
	case "master":
		var masterConfig *config.MasterConfig
		err = json.Unmarshal(cfgBuf, &masterConfig)
		if err != nil {
			l.WithError(err).Fatalln("unmarshal master configure file failed")
		}
		err = masterConfig.Validate()
		if err != nil {
			l.WithError(err).Fatalln("validate master configure file failed")
		}

		MasterMain(masterConfig)
	default:
		l.WithFields(log.Fields{
			"mode": baseCfg.Mode,
		}).Fatalln("unsupported mode")
	}

}
