package main

import (
	"io/ioutil"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func hints() {
	hintVMDirtyBackgroundRatio()
	hintVMDirtyExpireCentisecs()
}

func hintVMDirtyBackgroundRatio() {
	content, err := ioutil.ReadFile("/proc/sys/vm/dirty_background_ratio")
	if err != nil {
		return
	}

	value, err := strconv.ParseInt(strings.TrimRight(string(content), "\n"), 10, 32)
	if err != nil {
		return
	}
	if value == 10 {
		log.Info("You seem to use default value of vm.dirty_background_ratio. This parameter indicates how many percent of available memory can be used for holding dirty pages (pending writes). To save some I/O by background merging you might want to increase this value.")
	}
}

func hintVMDirtyExpireCentisecs() {
	content, err := ioutil.ReadFile("/proc/sys/vm/dirty_expire_centisecs")
	if err != nil {
		return
	}

	value, err := strconv.ParseInt(strings.TrimRight(string(content), "\n"), 10, 32)
	if err != nil {
		return
	}
	if value == 3000 {
		log.Info("You seem to use default value of vm.dirty_expire_centisecs. This parameter indicates maximum lifetime of dirty page before being forced to be written to disk (in centiseconds). To save some I/O by background merging you might want to increase this value.")
	}
}
