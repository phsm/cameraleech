package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type jsonCameraListReply struct {
	Data []jsonNameEntry `json:"data"`
}

type jsonNameEntry struct {
	Name string `json:"{#CAMERA}"`
}

func newRouter() *mux.Router {
	httpRouter := mux.NewRouter()
	httpRouter.HandleFunc("/cameras.json", zabbixAutodiscoveryCamerasListJSON)
	httpRouter.HandleFunc("/camera/{name}/frame", cameraFrame)
	httpRouter.HandleFunc("/camera/{name}/fps", cameraFps)
	httpRouter.HandleFunc("/camera/{name}/bitrate", cameraBitrate)
	httpRouter.HandleFunc("/camera/{name}/outtime", cameraOutTime)
	httpRouter.HandleFunc("/camera/{name}/dupframes", cameraDupFrames)
	httpRouter.HandleFunc("/camera/{name}/dropframes", cameraDropFrames)
	return httpRouter
}

func zabbixAutodiscoveryCamerasListJSON(w http.ResponseWriter, r *http.Request) {
	reply := jsonCameraListReply{}
	reply.Data = make([]jsonNameEntry, 0, 1024)

	for _, leech := range leeches {
		cam := jsonNameEntry{Name: leech.Config.Name}
		reply.Data = append(reply.Data, cam)
	}

	json, err := json.MarshalIndent(reply, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(json))
}

func cameraFrame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	camName := vars["name"]

	leech, ok := leeches[camName]
	if !ok {
		http.Error(w, fmt.Sprintf("Didn't find camera \"%s\"", camName), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d", leech.Stats.Frame)
}

func cameraFps(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	camName := vars["name"]

	leech, ok := leeches[camName]
	if !ok {
		http.Error(w, fmt.Sprintf("Didn't find camera \"%s\"", camName), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%f", leech.Stats.Fps)
}

func cameraBitrate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	camName := vars["name"]

	leech, ok := leeches[camName]
	if !ok {
		http.Error(w, fmt.Sprintf("Didn't find camera \"%s\"", camName), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d", leech.Stats.Bitrate)
}

func cameraOutTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	camName := vars["name"]

	leech, ok := leeches[camName]
	if !ok {
		http.Error(w, fmt.Sprintf("Didn't find camera \"%s\"", camName), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d", leech.Stats.OutTime/1000000)
}

func cameraDupFrames(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	camName := vars["name"]

	leech, ok := leeches[camName]
	if !ok {
		http.Error(w, fmt.Sprintf("Didn't find camera \"%s\"", camName), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d", leech.Stats.DupFrames)
}

func cameraDropFrames(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	camName := vars["name"]

	leech, ok := leeches[camName]
	if !ok {
		http.Error(w, fmt.Sprintf("Didn't find camera \"%s\"", camName), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d", leech.Stats.DropFrames)
}
