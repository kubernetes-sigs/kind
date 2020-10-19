/*
Copyright 2020 The FlowQ Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package server implements the `server` command
package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	_log "log"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/internal/runtime"
	"sigs.k8s.io/kind/pkg/log"
)

var _logger log.Logger
var provider *cluster.Provider

//initProvider build default provider struct
func initProvider(logger log.Logger) {

	provider = cluster.NewProvider(
		cluster.ProviderWithLogger(_logger),
		runtime.GetDefault(_logger),
	)
}

//creatCluster will call cluster.CreateWithRawConfig(rawConfig) create cluster
func createCluster(name string, rawConfig []byte) error {

	withConfig := cluster.CreateWithRawConfig(rawConfig)
	if len(rawConfig) == 0 {
		withConfig = cluster.CreateWithConfigFile("")
	}

	if err := provider.Create(
		name,
		withConfig,
		cluster.CreateWithNodeImage(""),
		cluster.CreateWithRetain(false),
		cluster.CreateWithWaitForReady(time.Duration(0)),
		cluster.CreateWithKubeconfigPath(""),
		cluster.CreateWithDisplayUsage(true),
		cluster.CreateWithDisplaySalutation(true),
	); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil

}

//handlerClusterList will list clusters
func handlerClusterList(w http.ResponseWriter, r *http.Request) {

	clusters, err := provider.List()
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"msg": fmt.Sprintf("list cluster failed, %", err.Error())})
		_logger.V(0).Infof("delete cluster failed, %", err.Error())
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"clusters": clusters})

}

//handlerClusterDelete will handle DELETE HTTP method, delete kubernetes cluster
func handlerClusterDelete(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	name := vars["name"]
	w.Header().Set("Content-Type", "application/json")
	if err := provider.Delete(name, ""); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"msg": fmt.Sprintf("delete %s cluster failed, %", name, err.Error())})
		_logger.V(0).Infof("delete %s cluster failed, %", name, err.Error())
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"msg": fmt.Sprintf("delete %s cluster successful", name)})
	_logger.V(0).Infof("delete %s cluster successful!", name)

}

//handlerClusterCreate process POST request, create kubernetes cluster
func handlerClusterCreate(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	name := vars["name"]
	if name == "" {
		name = "kind"
	}
	rawBody, err := ioutil.ReadAll(r.Body)
	if err == nil {
		_logger.V(0).Infof("%s\n", rawBody)
		go func() {
			_logger.V(0).Infof("creating [%s] cluster ", name)
			err := createCluster(name, rawBody)
			if err != nil {
				_logger.Error(err.Error())
			} else {
				_logger.V(0).Infof("create [%s] completed", name)
			}
		}()
	} else {
		_logger.Error(err.Error())
	}

	err = json.NewEncoder(w).Encode(map[string]string{"msg": fmt.Sprintf("creating %s cluster.....", name)})
	if err != nil {
		_logger.Error(err.Error())
	}

}

//APIServerStart is a API Server main func
func APIServerStart(logger log.Logger, address, port string) {

	_logger = logger

	initProvider(logger)

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/cluster/{name}", handlerClusterCreate).Methods("POST")
	r.HandleFunc("/api/v1/cluster/{name}", handlerClusterDelete).Methods("DELETE")
	r.HandleFunc("/api/v1/clusters", handlerClusterList).Methods("GET")

	r.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(map[string]string{"server": "kind"})
		if err != nil {
			_log.Print(err)
		}
	})

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf("%s:%s", address, port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	_log.Printf("APIServer Listen : %s", fmt.Sprintf("%s:%s", address, port))
	_log.Fatal(srv.ListenAndServe())

}
