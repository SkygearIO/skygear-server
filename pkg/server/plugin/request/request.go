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

package request

import (
	"encoding/json"

	skyplugin "github.com/skygeario/skygear-server/pkg/server/plugin"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	"github.com/skygeario/skygear-server/pkg/server/skydb/skyconv"
	"golang.org/x/net/context"
)

// Request represents data in a server to worker plugin request.
type Request struct {
	Context context.Context
	Kind    string
	Name    string
	Param   interface{}
}

// HookRequest contains records involved in a database hook.
type HookRequest struct {
	Record   interface{} `json:"record"`
	Original interface{} `json:"original"`
}

// NewLambdaRequest creates a new lambda request.
func NewLambdaRequest(ctx context.Context, name string, args json.RawMessage) *Request {
	return &Request{Kind: "op", Name: name, Param: args, Context: ctx}
}

// NewHandlerRequest creates a new handler request.
func NewHandlerRequest(ctx context.Context, name string, input json.RawMessage) *Request {
	return &Request{Kind: "handler", Name: name, Param: input, Context: ctx}
}

// NewHookRequest creates a new hook request.
func NewHookRequest(ctx context.Context, hookName string, record *skydb.Record, originalRecord *skydb.Record) *Request {
	param := HookRequest{
		Record:   (*skyconv.JSONRecord)(record),
		Original: (*skyconv.JSONRecord)(originalRecord),
	}
	return &Request{Kind: "hook", Name: hookName, Param: param, Context: ctx}
}

// NewAuthRequest creates a new auth request.
func NewAuthRequest(authReq *skyplugin.AuthRequest) *Request {
	return &Request{
		Kind: "provider",
		Name: authReq.ProviderName,
		Param: struct {
			Action   string                 `json:"action"`
			AuthData map[string]interface{} `json:"auth_data"`
		}{authReq.Action, authReq.AuthData},
	}
}

// MarshalJSON converts a request to JSON representation.
func (req *Request) MarshalJSON() ([]byte, error) {
	// TODO(limouren): reduce copying of this method
	pluginCtx := skyplugin.ContextMap(req.Context)
	if rawParam, ok := req.Param.(json.RawMessage); ok {
		rawParamReq := struct {
			Kind    string                 `json:"kind"`
			Name    string                 `json:"name,omitempty"`
			Param   json.RawMessage        `json:"param,omitempty"`
			Context map[string]interface{} `json:"context,omitempty"`
		}{req.Kind, req.Name, rawParam, pluginCtx}
		return json.Marshal(&rawParamReq)
	}

	paramReq := struct {
		Kind    string                 `json:"kind"`
		Name    string                 `json:"name,omitempty"`
		Param   interface{}            `json:"param,omitempty"`
		Context map[string]interface{} `json:"context,omitempty"`
	}{req.Kind, req.Name, req.Param, pluginCtx}

	return json.Marshal(&paramReq)
}
