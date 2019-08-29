package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-cmd/cmd"
	log "github.com/sirupsen/logrus"
)

var (
	leeches map[string]*leech
)

type leech struct {
	Config cameraConfig

	stopEverything   bool // If set to true, everything will stop
	stopEverythingMu sync.Mutex
	watcherStarted   bool
	stopWatcher      chan int

	progMsgsCounter     int
	progMsgsStringsPool []string
	progMsgsPool        []progressMessage
	Stats               progressMessage

	command *cmd.Cmd
	status  <-chan cmd.Status
}

type progressMessage struct {
	Frame      uint64
	Fps        float32
	Bitrate    int
	OutTime    uint64 // miliseconds
	DupFrames  int
	DropFrames int
}

func newLeech(c cameraConfig) *leech {
	l := new(leech)
	l.Config = cameraConfig{}
	l.Config = c
	l.Stats = progressMessage{}
	l.stopWatcher = make(chan int, 1)
	l.progMsgsStringsPool = make([]string, 0, 128)
	l.progMsgsPool = make([]progressMessage, 0, 128)
	return l
}

func (l *leech) Start() error {
	var ffmpegArgs []string
	var err error
	l.setStopEverythingValue(false)

	filePath := fmt.Sprintf("%s/%s/%%Y-%%m-%%d/%%Y-%%m-%%d_%%H-%%M-%%S.mkv", l.Config.StoragePath, l.Config.Name)
	ffmpegArgs = make([]string, 0, 10)

	log.Debugf("Stream %s: Assembling ffmpeg command", l.Config.Name)
	ffmpegArgs = append(ffmpegArgs, "-hide_banner", "-nostdin", "-nostats", "-progress", "pipe:1",
		"-loglevel", l.Config.FfmpegLogLevel, "-i", l.Config.URL, "-codec", "copy",
		"-f", "segment", "-segment_time", fmt.Sprint(l.Config.SegmentTime), "-reset_timestamps", "1",
		"-segment_atclocktime", "1", "-strftime", "1", filePath)

	log.Debug("Creating necessary subfolders (if needed)")
	if err := l.createSubFolders(); err != nil {
		log.Errorf("Error creating subfolder for camera %s segments: %v", l.Config.Name, err)
		return err
	}

	l.command = cmd.NewCmdOptions(cmd.Options{Streaming: true}, l.Config.FfmpegPath, ffmpegArgs...)
	l.status = l.command.Start()
	log.Infof("Camera %s: ffmpeg leech started", l.Config.Name)

	if !l.watcherStarted {
		// Starting crash watcher if the program has been started
		log.Debugf("Camera %s: starting watcher", l.Config.Name)
		go func() {
			l.watcherStarted = true
			for {
				log.Debugf("Camera %s: waiting watcher channel to send event", l.Config.Name)
				<-l.status
				if l.getStopEverythingValue() {
					l.watcherStarted = false
					return
				}

				l.stopWatcher <- 666

				if err != nil {
					log.Errorf("Camera %s: command was finished with error: %v", l.Config.Name, err)
				}

				time.Sleep(1 * time.Second)

				log.Infof("Camera %s: restarting ffmpeg command with args %v", l.Config.Name, ffmpegArgs)
				l.Start()
			}
		}()
	}

	// Starting routine creating folders for the next day
	go func() {
		for {
			if l.getStopEverythingValue() {
				return
			}

			t := time.Now()
			hour := t.Hour()
			minute := t.Minute()
			if hour == 23 && minute > 50 && minute < 56 {
				if err := l.createNextDaySubfolders(); err != nil {
					log.Errorf("Error creating next-day subfolder for camera %s: %v", l.Config.Name, err)
				}
			}
			time.Sleep(295 * time.Second)
		}
	}()

	// Starting a goroutine which will be grabbing strings from stdout and stderr chans
	log.Debugf("Camera %s: Starting output grabber", l.Config.Name)
	go func() {
		for {
			if l.getStopEverythingValue() {
				return
			}

			select {
			case stdout := <-l.command.Stdout:
				l.handleProgressMessage(stdout)
			case stderr := <-l.command.Stderr:
				l.sendLog(stderr)
			case <-l.stopWatcher:
				log.Infof("Camera %s: program output grabber routine destroyed", l.Config.Name)
				return
			}
		}
	}()

	return nil
}

func (l *leech) setStopEverythingValue(val bool) {
	l.stopEverythingMu.Lock()
	defer l.stopEverythingMu.Unlock()
	l.stopEverything = val
}

func (l *leech) getStopEverythingValue() bool {
	l.stopEverythingMu.Lock()
	defer l.stopEverythingMu.Unlock()
	return l.stopEverything
}

func (l *leech) createSubFolders() error {
	t := time.Now()
	dateString := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
	path := fmt.Sprintf("%s/%s/%s", l.Config.StoragePath, l.Config.Name, dateString)
	return os.MkdirAll(path, 0755)
}

func (l *leech) createNextDaySubfolders() error {
	t := time.Now().AddDate(0, 0, 1)
	dateString := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
	path := fmt.Sprintf("%s/%s/%s", l.Config.StoragePath, l.Config.Name, dateString)
	return os.MkdirAll(path, 0755)
}

func (l *leech) Stop() error {
	log.Infof("Camera %s: stopping", l.Config.Name)
	l.setStopEverythingValue(true)
	err := l.command.Stop()
	if err != nil {
		log.Errorf("Camera %s: error during command stop: %v", l.Config.Name, err)
		return err
	}
	return nil
}

func (l *leech) handleProgressMessage(str string) {
	possibleValues := []string{"frame=", "fps=", "bitrate=", "out_time_ms=", "dup_frames=", "drop_frames="}
	if strings.Contains(str, "progress=") {
		msg := progressMessage{}

		for _, s := range l.progMsgsStringsPool {
			if strings.Contains(s, "frame=") {
				// handle "frame=1234" string
				framecountStr := strings.TrimLeft(s, "frame=")
				frame, err := strconv.ParseUint(framecountStr, 10, 64)
				if err != nil {
					log.Errorf("Failed to parse Uint in the string: \"%s\": %v", framecountStr, err)
					continue
				}
				msg.Frame = frame
			} else if strings.Contains(s, "fps=") {
				// handle fps
				fpsStr := strings.TrimLeft(s, "fps=")
				fpsVal, err := strconv.ParseFloat(fpsStr, 32)
				if err != nil {
					log.Errorf("Failed to parse Float in the string: \"%s\": %v", fpsStr, err)
					continue
				}
				msg.Fps = float32(fpsVal)
			} else if strings.Contains(s, "bitrate=") {
				bitrateStr := strings.TrimLeft(s, "bitrate=")

				// Put -1 instead of bitrate if ffmpeg can't give it to us
				var bitrateVal int64
				if strings.Contains(bitrateStr, "N/A") {
					bitrateVal = -1
				} else {
					var err error
					bitrateSlice := strings.Split(bitrateStr, ".")
					bitrateVal, err = strconv.ParseInt(bitrateSlice[0], 10, 0)
					if err != nil {
						log.Errorf("Failed to parse Int in the string: \"%s\": %v", bitrateSlice[0], err)
						continue
					}
				}
				msg.Bitrate = int(bitrateVal)
			} else if strings.Contains(s, "out_time_ms=") {
				outTimeStr := strings.TrimLeft(s, "out_time_ms=")
				outtime, err := strconv.ParseUint(outTimeStr, 10, 64)
				if err != nil {
					log.Errorf("Failed to parse Uint in the string: \"%s\": %v", outTimeStr, err)
					continue
				}
				msg.OutTime = outtime
			} else if strings.Contains(s, "dup_frames=") {
				dupFramesStr := strings.TrimLeft(s, "dup_frames=")
				dupFramesVal, err := strconv.ParseInt(dupFramesStr, 10, 0)
				if err != nil {
					log.Errorf("Failed to parse Int in the string: \"%s\": %v", dupFramesStr, err)
					continue
				}
				msg.DupFrames = int(dupFramesVal)
			} else if strings.Contains(s, "drop_frames=") {
				dropFramesStr := strings.TrimLeft(s, "drop_frames=")
				dropFramesVal, err := strconv.ParseInt(dropFramesStr, 10, 0)
				if err != nil {
					log.Errorf("Failed to parse Int in the string: \"%s\": %v", dropFramesStr, err)
					continue
				}
				msg.DropFrames = int(dropFramesVal)
			}
		}

		l.progMsgsStringsPool = nil

		l.progMsgsPool = append(l.progMsgsPool, msg)
		l.progMsgsCounter++
		if l.progMsgsCounter > 60 {
			l.sendProgressReport()
			l.progMsgsPool = nil
			l.progMsgsCounter = 0
		}
	} else {
		for _, val := range possibleValues {
			if strings.Contains(str, val) {
				l.progMsgsStringsPool = append(l.progMsgsStringsPool, str)
			}
		}
	}
}

func (l *leech) sendProgressReport() {
	var avgFPSTemp float32
	var biteateAvgTemp int

	lastIdx := len(l.progMsgsPool) - 1
	l.Stats.Frame = l.progMsgsPool[lastIdx].Frame
	l.Stats.OutTime = l.progMsgsPool[lastIdx].OutTime
	l.Stats.DupFrames = l.progMsgsPool[lastIdx].DupFrames
	l.Stats.DropFrames = l.progMsgsPool[lastIdx].DropFrames

	for _, i := range l.progMsgsPool {
		avgFPSTemp = avgFPSTemp + i.Fps
		biteateAvgTemp = biteateAvgTemp + i.Bitrate
	}

	l.Stats.Fps = avgFPSTemp / float32(len(l.progMsgsPool))
	l.Stats.Bitrate = biteateAvgTemp / len(l.progMsgsPool)
}

func (l *leech) sendLog(str string) {
	log.Infof("%s ffmpeg output: %s", l.Config.Name, str)
}
