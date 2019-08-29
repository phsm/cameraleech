package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

var (
	configPath string

	configMu sync.Mutex
	config   tomlConfig
)

type tomlConfig struct {
	httpListenAddress string
	Loglevel          string
	DisableHints      bool
	Defaults          cameraConfig
	Cameras           map[string]cameraConfig
}

type cameraConfig struct {
	Name           string
	FfmpegPath     string
	FfmpegLogLevel string
	StoragePath    string
	SegmentTime    int
	URL            string
}

func parseFlags() {
	flag.StringVar(&configPath, "config", "/etc/cameraleech.toml", "Path to the config file")
	flag.Parse()
}

func identicalConfigAndLeech(s cameraConfig, l *leech) bool {
	if s == l.Config {
		return true
	}
	return false
}

func launchLeeches() error {
	// check if leeches map is allocated
	if leeches == nil {
		leeches = make(map[string]*leech)
	}

	// First, check if we need to delete some leeches by id
	for k := range leeches {
		toDelete := true
		for j := range config.Cameras {
			if k == j {
				toDelete = false
				break
			}
		}

		if !toDelete {
			continue
		}

		if err := leeches[k].Stop(); err != nil {
			log.Errorf("Error stopping camera %s: %s", k, err)
			continue
		}
		delete(leeches, k)

	}

	// Then, we need to launch new streams
	for k, val := range config.Cameras {
		toAdd := true
		for j := range leeches {
			if k == j {
				toAdd = false
				break
			}
		}
		if !toAdd {
			continue
		}

		l := newLeech(config.Cameras[k])
		l.Config = val

		if err := l.Start(); err != nil {
			log.Errorf("Error starting camera %s: %v", k, err)
			continue
		}
		leeches[k] = l
	}

	// Check if we need to reconfigure some existing leeches
	for k, val := range leeches {
		s, ok := config.Cameras[k]
		if !ok {
			log.Warnf("Failed to look camera %s in config. That's strange, it should be there. Skipping.", k)
			continue
		}
		if identicalConfigAndLeech(s, val) {
			continue
		}
		val.Config = s
		val.Stop()
		val.Start()
	}
	return nil
}

func readConfig(filePath string) error {
	configMu.Lock()
	defer configMu.Unlock()

	configText, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	config = tomlConfig{}
	if _, err := toml.Decode(string(configText), &config); err != nil {
		return err
	}

	if config.httpListenAddress == "" {
		config.httpListenAddress = "127.0.0.1:8080"
	}

	// выставляем уровень логирования
	switch config.Loglevel {
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "":
		log.SetLevel(log.WarnLevel)
	default:
		return errors.New("Log level must be one of the following: fatal, error, warn, info, debug")
	}

	if config.Defaults.FfmpegLogLevel == "" {
		config.Defaults.FfmpegLogLevel = "repeat+level+error"
	}

	// Default segment time is 3600 seconds (1 hour)
	if config.Defaults.SegmentTime == 0 {
		config.Defaults.SegmentTime = 3600
	}

	// Расставляем дефолтные значения если не указано в камерах
	for camName := range config.Cameras {
		camConfig := config.Cameras[camName]
		camConfig.Name = camName

		if camConfig.FfmpegPath == "" {
			camConfig.FfmpegPath = config.Defaults.FfmpegPath
		}

		if camConfig.FfmpegLogLevel == "" {
			camConfig.FfmpegLogLevel = config.Defaults.FfmpegLogLevel
		}

		if camConfig.StoragePath == "" {
			camConfig.StoragePath = config.Defaults.StoragePath
		}

		if camConfig.StoragePath == "" {
			return fmt.Errorf("Camera %s: storage path must not be empty. Either specify the camera or default storage path", camName)
		}

		if camConfig.SegmentTime == 0 {
			camConfig.SegmentTime = config.Defaults.SegmentTime
		}

		if camConfig.URL == "" {
			return fmt.Errorf("You have to specify URL for camera %s", camConfig.Name)
		}

		config.Cameras[camName] = camConfig
	}
	return nil
}

func signalWatcher(c chan os.Signal) {
	for {
		s := <-c
		switch s {
		case syscall.SIGTERM, syscall.SIGINT:
			log.Info("Got termination signal")
			programIsStopping = true
			cond.Signal()

			for k, l := range leeches {
				log.Infof("Terminating camera %s", k)
				l.Stop()
				cond.L.Lock()
				delete(leeches, k)
				cond.L.Unlock()
				cond.Signal()
			}
		case syscall.SIGHUP:
			log.Info("Reloading configuration")
			if err := readConfig(configPath); err != nil {
				log.Warnf("Error reloading configuration: %v", err)
				break
			}
			if err := launchLeeches(); err != nil {
				log.Warnf("Error (re-)launching leeches: %v", err)
				break
			}
		}
	}
}
