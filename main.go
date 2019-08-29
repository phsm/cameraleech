package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var (
	cond              *sync.Cond
	commit            string // value is provided in compilation phase (see Makefile)
	builtat           string // value is provided in compilation phase (see Makefile)
	programIsStopping bool   // if the main program is stopping, it should be set to true
	httpServer        *http.Server
)

func main() {
	parseFlags()

	log.Infof("Cameraleech commit %s, built at %s, %s started", commit, builtat, runtime.Version())

	// subscribing on SIGINT, SIGTERM - graceful shutdown of ffmpegs
	// Also SIGHUP for reload
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalWatcher(signalChan)

	if err := readConfig(configPath); err != nil {
		log.Fatalf("Can not read config: %s", err)
	}

	// print hints
	hints()

	err := launchLeeches()
	if err != nil {
		log.Errorf("Couldn't launch camera leeches: %v", err)
	}
	log.Info("Camera leeches are launched")

	httpRouter := newRouter()
	http.Handle("/", httpRouter)

	httpServer := &http.Server{Addr: config.httpListenAddress, Handler: httpRouter}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Error listening http stats server: %v", err)
		}
	}()

	cond = &sync.Cond{L: &sync.Mutex{}}
	cond.L.Lock()
	for !programIsStopping || len(leeches) > 0 {
		cond.Wait()
	}
	cond.L.Unlock()
}
