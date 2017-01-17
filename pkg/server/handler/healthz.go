// Copyright 2015-present Oursky Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"github.com/skygeario/skygear-server/pkg/server/router"
)

type healthStatusResponse struct {
	Status string `json:"status,omitempty"`
}

type HealthzHandler struct {
	PluginReady   router.Processor `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

func (h *HealthzHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.PluginReady,
	}
}

func (h *HealthzHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

func (h *HealthzHandler) Handle(playload *router.Payload, response *router.Response) {
	var (
		rep healthStatusResponse
	)
	rep.Status = "OK"
	response.Result = rep
	return
}
