package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/go-zoo/bone"
)

// ContainerInfo stores information about containers
type ContainerInfo struct {
	Pid         int               `json:"pid"`
	Image       string            `json:"image"`
	CreatedTime int64             `json:"created_time"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	LogPath     string            `json:"log_path"`
	Root        string            `json:"root"`
	Sandbox     string            `json:"sandbox"`
}

// CrioInfo stores information about the crio daemon
type CrioInfo struct {
	StorageDriver string `json:"storage_driver"`
	StorageRoot   string `json:"storage_root"`
	CgroupDriver  string `json:"cgroup_driver"`
}

// GetInfoMux returns the mux used to serve info requests
func (s *Server) GetInfoMux() *bone.Mux {
	mux := bone.New()

	mux.Get("/info", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ci := CrioInfo{
			StorageDriver: s.config.Config.Storage,
			StorageRoot:   s.config.Config.Root,
			CgroupDriver:  s.config.Config.CgroupManager,
		}
		js, err := json.Marshal(ci)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))

	mux.Get("/containers/:id", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		containerID := bone.GetValue(req, "id")
		ctr := s.GetContainer(containerID)
		if ctr == nil {
			ctr = s.getInfraContainer(containerID)
			if ctr == nil {
				http.Error(w, fmt.Sprintf("container with id: %s not found", containerID), http.StatusNotFound)
				return
			}
		}
		ctrState := ctr.State()
		if ctrState == nil {
			http.Error(w, fmt.Sprintf("container %s state is nil", containerID), http.StatusNotFound)
			return
		}

		ci := ContainerInfo{
			Pid:         ctrState.Pid,
			Image:       ctr.Image(),
			CreatedTime: ctrState.Created.UnixNano(),
			Labels:      ctr.Labels(),
			Annotations: ctr.Annotations(),
			Root:        ctr.MountPoint(),
			LogPath:     filepath.Dir(ctr.LogPath()),
			Sandbox:     ctr.Sandbox(),
		}
		js, err := json.Marshal(ci)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))

	return mux
}
