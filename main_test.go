package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConfig               = "tests/goodconfig.toml"
	testConfigDeleteCam      = "tests/goodconfig_deletecam.toml"
	testConfigAddCam         = "tests/goodconfig_addcam.toml"
	testConfigBadLogLevel    = "tests/badconfig_loglevel.toml"
	testConfigBadStoragePath = "tests/badconfig_storagepath.toml"
	testConfigMissingURL     = "tests/badconfig_missingurl.toml"
)

func deleteDownloadedData(t *testing.T, path string) {
	err := os.RemoveAll(path)
	assert.Nil(t, err)
}

func stopLeeches(t *testing.T) {
	for k, l := range leeches {
		err := l.Stop()
		assert.Nil(t, err)

		delete(leeches, k)
	}
}

func TestLaunchLeeches(t *testing.T) {
	var err error
	err = readConfig(testConfig)
	require.Nil(t, err)

	defer deleteDownloadedData(t, config.Defaults.StoragePath)

	err = launchLeeches()
	require.Nil(t, err)

	time.Sleep(20 * time.Second)

	err = readConfig(testConfigDeleteCam)
	require.Nil(t, err)

	err = launchLeeches()
	require.Nil(t, err)

	time.Sleep(30 * time.Second)

	err = readConfig(testConfigAddCam)
	require.Nil(t, err)

	err = launchLeeches()
	require.Nil(t, err)

	time.Sleep(30 * time.Second)

	stopLeeches(t)
}

func TestStats(t *testing.T) {
	var err error
	err = readConfig(testConfig)
	require.Nil(t, err)

	defer deleteDownloadedData(t, config.Defaults.StoragePath)

	err = launchLeeches()
	require.Nil(t, err)

	defer stopLeeches(t)

	time.Sleep(65 * time.Second)

	router := newRouter()

	// Zabbix autodiscovery test
	req := httptest.NewRequest("GET", "/cameras.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	reply := new(jsonCameraListReply)
	cameraList := make([]string, 0, 2)
	err = json.Unmarshal(body, reply)
	require.Nil(t, err)

	for _, v := range reply.Data {
		cameraList = append(cameraList, v.Name)
	}

	assert.Equal(t, 200, resp.StatusCode)
	assert.Subset(t, []string{"cam1", "cam2"}, cameraList)

	// Camera Frame number
	req = httptest.NewRequest("GET", "/camera/cam1/frame", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	fmt.Println(string(body))
	frame, err := strconv.ParseUint(string(body), 10, 64)
	require.Nil(t, err)

	assert.Greater(t, frame, uint64(0))

	// Camera FPS
	req = httptest.NewRequest("GET", "/camera/cam1/fps", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	fps, err := strconv.ParseFloat(string(body), 10)
	require.Nil(t, err)

	assert.Greater(t, fps, 0.0)

	// Camera bitrate
	req = httptest.NewRequest("GET", "/camera/cam1/bitrate", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	_, err = strconv.ParseInt(string(body), 10, 32)
	require.Nil(t, err)

	// Camera outtime
	req = httptest.NewRequest("GET", "/camera/cam1/outtime", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	outtime, err := strconv.ParseUint(string(body), 10, 64)
	require.Nil(t, err)

	assert.Greater(t, outtime, uint64(0))

	// Camera dupframes
	req = httptest.NewRequest("GET", "/camera/cam1/dupframes", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	_, err = strconv.ParseInt(string(body), 10, 32)
	require.Nil(t, err)

	// Camera dropframes
	req = httptest.NewRequest("GET", "/camera/cam1/dropframes", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp = w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	_, err = strconv.ParseInt(string(body), 10, 32)
	require.Nil(t, err)

}

func TestBadConfig(t *testing.T) {
	var err error
	err = readConfig(testConfigBadLogLevel)
	require.NotNil(t, err)

	err = readConfig(testConfigBadStoragePath)
	require.NotNil(t, err)

	err = readConfig(testConfigMissingURL)
	require.NotNil(t, err)
}
