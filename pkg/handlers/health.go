// Copyright 2018 The Gardener Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"net/http"
	"sync"
)

var (
	mutex   sync.Mutex
	healthy = false
)

// UpdateHealth expects a boolean value <isHealthy> and assigns it to the package-internal 'healthy' variable.
func UpdateHealth(isHealthy bool) {
	mutex.Lock()
	healthy = isHealthy
	mutex.Unlock()
}

// Healthz is an HTTP handler for the /healthz endpoint which responds with 200 OK status code
// if the Machine Controller Manager is healthy; and with 500 Internal Server error status code otherwise.
func Healthz(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	isHealthy := healthy
	mutex.Unlock()
	if isHealthy {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		runtime.Must(err)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	_, err := w.Write([]byte("Unhealthy"))
	runtime.Must(err)
}
