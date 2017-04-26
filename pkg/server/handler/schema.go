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
	"strings"

	"github.com/mitchellh/mapstructure"
	pluginEvent "github.com/skygeario/skygear-server/pkg/server/plugin/event"
	"github.com/skygeario/skygear-server/pkg/server/router"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	"github.com/skygeario/skygear-server/pkg/server/skydb/skyconv"
	"github.com/skygeario/skygear-server/pkg/server/skyerr"
)

type schemaResponse struct {
	Schemas map[string]schemaFieldList `json:"record_types"`
}

/*
SchemaRenameHandler handles the action of renaming column
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/rename <<EOF
{
	"master_key": "MASTER_KEY",
	"action": "schema:rename",
	"record_type": "student",
	"item_type": "field",
	"item_name": "score",
	"new_name": "exam_score"
}
EOF
*/
type SchemaRenameHandler struct {
	EventSender   pluginEvent.Sender `inject:"PluginEventSender"`
	AccessKey     router.Processor   `preprocessor:"accesskey"`
	DevOnly       router.Processor   `preprocessor:"dev_only"`
	DBConn        router.Processor   `preprocessor:"dbconn"`
	InjectDB      router.Processor   `preprocessor:"inject_db"`
	PluginReady   router.Processor   `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

func (h *SchemaRenameHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.AccessKey,
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
		h.PluginReady,
	}
}

func (h *SchemaRenameHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

type schemaRenamePayload struct {
	RecordType string `mapstructure:"record_type"`
	OldName    string `mapstructure:"item_name"`
	NewName    string `mapstructure:"new_name"`
}

func (payload *schemaRenamePayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}
	return payload.Validate()
}

func (payload *schemaRenamePayload) Validate() skyerr.Error {
	missingArgs := []string{}
	if payload.RecordType == "" {
		missingArgs = append(missingArgs, "record_type")
	}
	if payload.OldName == "" {
		missingArgs = append(missingArgs, "item_name")
	}
	if payload.NewName == "" {
		missingArgs = append(missingArgs, "new_name")
	}
	if len(missingArgs) > 0 {
		return skyerr.NewInvalidArgument("missing required fields", missingArgs)
	}
	if strings.HasPrefix(payload.RecordType, "_") {
		return skyerr.NewInvalidArgument("attempts to change reserved table", []string{"record_type"})
	}
	if strings.HasPrefix(payload.OldName, "_") {
		return skyerr.NewInvalidArgument("attempts to change reserved key", []string{"item_name"})
	}
	if strings.HasPrefix(payload.NewName, "_") {
		return skyerr.NewInvalidArgument("attempts to change reserved key", []string{"new_name"})
	}
	return nil
}

func (h *SchemaRenameHandler) Handle(rpayload *router.Payload, response *router.Response) {
	payload := &schemaRenamePayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	db := rpayload.Database
	if err := db.RenameSchema(payload.RecordType, payload.OldName, payload.NewName); err != nil {
		response.Err = skyerr.NewError(skyerr.ResourceNotFound, err.Error())
		return
	}

	schemas, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.MakeError(err)
		return
	}
	response.Result = &schemaResponse{
		Schemas: encodeRecordSchemas(schemas),
	}

	if h.EventSender != nil {
		err := sendSchemaChangedEvent(h.EventSender, db)
		if err != nil {
			log.WithField("err", err).Warn("Fail to send schema changed event")
		}
	}
}

/*
SchemaDeleteHandler handles the action of deleting column
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/delete <<EOF
{
	"master_key": "MASTER_KEY",
	"action": "schema:delete",
	"record_type": "student",
	"item_name": "score"
}
EOF
*/
type SchemaDeleteHandler struct {
	EventSender   pluginEvent.Sender `inject:"PluginEventSender"`
	AccessKey     router.Processor   `preprocessor:"accesskey"`
	DevOnly       router.Processor   `preprocessor:"dev_only"`
	DBConn        router.Processor   `preprocessor:"dbconn"`
	InjectDB      router.Processor   `preprocessor:"inject_db"`
	PluginReady   router.Processor   `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

func (h *SchemaDeleteHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.AccessKey,
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
		h.PluginReady,
	}
}

func (h *SchemaDeleteHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

type schemaDeletePayload struct {
	RecordType string `mapstructure:"record_type"`
	ColumnName string `mapstructure:"item_name"`
}

func (payload *schemaDeletePayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}
	return payload.Validate()
}

func (payload *schemaDeletePayload) Validate() skyerr.Error {
	missingArgs := []string{}
	if payload.RecordType == "" {
		missingArgs = append(missingArgs, "record_type")
	}
	if payload.ColumnName == "" {
		missingArgs = append(missingArgs, "item_name")
	}
	if len(missingArgs) > 0 {
		return skyerr.NewInvalidArgument("missing required fields", missingArgs)
	}
	if strings.HasPrefix(payload.RecordType, "_") {
		return skyerr.NewInvalidArgument("attempts to change reserved table", []string{"record_type"})
	}

	if strings.HasPrefix(payload.ColumnName, "_") {
		return skyerr.NewInvalidArgument("attempts to change reserved key", []string{"item_name"})
	}
	return nil
}

func (h *SchemaDeleteHandler) Handle(rpayload *router.Payload, response *router.Response) {
	payload := &schemaDeletePayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	db := rpayload.Database
	if err := db.DeleteSchema(payload.RecordType, payload.ColumnName); err != nil {
		response.Err = skyerr.NewError(skyerr.ResourceNotFound, err.Error())
		return
	}

	schemas, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.MakeError(err)
		return
	}
	response.Result = &schemaResponse{
		Schemas: encodeRecordSchemas(schemas),
	}

	if h.EventSender != nil {
		err := sendSchemaChangedEvent(h.EventSender, db)
		if err != nil {
			log.WithField("err", err).Warn("Fail to send schema changed event")
		}
	}
}

/*
SchemaCreateHandler handles the action of creating new columns
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/create <<EOF
{
	"master_key": "MASTER_KEY",
	"action": "schema:create",
	"record_types":{
		"student": {
			"fields":[
				{"name": "age", "type": "number"},
				{"name": "nickname" "type": "string"}
			]
		}
	}
}
EOF
*/
type SchemaCreateHandler struct {
	EventSender   pluginEvent.Sender `inject:"PluginEventSender"`
	AccessKey     router.Processor   `preprocessor:"accesskey"`
	DevOnly       router.Processor   `preprocessor:"dev_only"`
	DBConn        router.Processor   `preprocessor:"dbconn"`
	InjectDB      router.Processor   `preprocessor:"inject_db"`
	PluginReady   router.Processor   `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

func (h *SchemaCreateHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.AccessKey,
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
		h.PluginReady,
	}
}

func (h *SchemaCreateHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

type schemaCreatePayload struct {
	RawSchemas map[string]schemaFieldList `mapstructure:"record_types"`

	Schemas map[string]skydb.RecordSchema
}

func (payload *schemaCreatePayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}

	payload.Schemas = make(map[string]skydb.RecordSchema)
	for recordType, schema := range payload.RawSchemas {
		payload.Schemas[recordType] = make(skydb.RecordSchema)
		for _, field := range schema.Fields {
			var err error
			payload.Schemas[recordType][field.Name], err = skydb.SimpleNameToFieldType(field.TypeName)
			if err != nil {
				return skyerr.NewInvalidArgument("unexpected field type", []string{field.TypeName})
			}
		}
	}

	return payload.Validate()
}

func (payload *schemaCreatePayload) Validate() skyerr.Error {
	for recordType, schema := range payload.Schemas {
		if strings.HasPrefix(recordType, "_") {
			return skyerr.NewInvalidArgument("attempts to create reserved table", []string{recordType})
		}
		for fieldName := range schema {
			if strings.HasPrefix(fieldName, "_") {
				return skyerr.NewInvalidArgument("attempts to create reserved field", []string{fieldName})
			}
		}
	}
	return nil
}

func (h *SchemaCreateHandler) Handle(rpayload *router.Payload, response *router.Response) {
	log.Debugf("%+v\n", rpayload)

	payload := &schemaCreatePayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	db := rpayload.Database
	for recordType, recordSchema := range payload.Schemas {
		_, err := db.Extend(recordType, recordSchema)
		if err != nil {
			response.Err = skyerr.NewError(skyerr.IncompatibleSchema, err.Error())
			return
		}
	}

	schemas, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.MakeError(err)
		return
	}
	response.Result = &schemaResponse{
		Schemas: encodeRecordSchemas(schemas),
	}

	if h.EventSender != nil {
		err := sendSchemaChangedEvent(h.EventSender, db)
		if err != nil {
			log.WithField("err", err).Warn("Fail to send schema changed event")
		}
	}
}

/*
SchemaFetchHandler handles the action of returing information of record schema
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/fetch <<EOF
{
	"master_key": "MASTER_KEY",
	"action": "schema:fetch"
}
EOF
*/
type SchemaFetchHandler struct {
	AccessKey     router.Processor `preprocessor:"accesskey"`
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	PluginReady   router.Processor `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

func (h *SchemaFetchHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.AccessKey,
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
		h.PluginReady,
	}
}

func (h *SchemaFetchHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

func (h *SchemaFetchHandler) Handle(rpayload *router.Payload, response *router.Response) {
	db := rpayload.Database
	schemas, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.MakeError(err)
		return
	}

	response.Result = &schemaResponse{
		Schemas: encodeRecordSchemas(schemas),
	}
}

/*
SchemaAccessHandler handles the update of creation access of record
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/access <<EOF
{
	"master_key": "MASTER_KEY",
	"action": "schema:access",
	"type": "note",
	"create_roles": [
		"admin",
		"writer"
	]
}
EOF
*/
type SchemaAccessHandler struct {
	AccessKey     router.Processor `preprocessor:"accesskey"`
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	PluginReady   router.Processor `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

type schemaAccessPayload struct {
	Type           string   `mapstructure:"type"`
	RawCreateRoles []string `mapstructure:"create_roles"`
	ACL            skydb.RecordACL
}

type schemaAccessResponse struct {
	Type        string   `json:"type"`
	CreateRoles []string `json:"create_roles,omitempty"`
}

func (h *SchemaAccessHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.AccessKey,
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
		h.PluginReady,
	}
}

func (h *SchemaAccessHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

func (payload *schemaAccessPayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}

	acl := skydb.RecordACL{}
	for _, perRoleName := range payload.RawCreateRoles {
		acl = append(acl, skydb.NewRecordACLEntryRole(perRoleName, skydb.CreateLevel))
	}

	payload.ACL = acl

	return payload.Validate()
}

func (payload *schemaAccessPayload) Validate() skyerr.Error {
	if payload.Type == "" {
		return skyerr.NewInvalidArgument("missing required fields", []string{"type"})
	}

	return nil
}

func (h *SchemaAccessHandler) Handle(rpayload *router.Payload, response *router.Response) {
	payload := schemaAccessPayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	c := rpayload.Database.Conn()
	err := c.SetRecordAccess(payload.Type, payload.ACL)

	if err != nil {
		if skyErr, isSkyErr := err.(skyerr.Error); isSkyErr {
			response.Err = skyErr
		} else {
			response.Err = skyerr.MakeError(err)
		}
		return
	}

	response.Result = schemaAccessResponse{
		Type:        payload.Type,
		CreateRoles: payload.RawCreateRoles,
	}
}

/*
SchemaDefaultAccessHandler handles the update of creation access of record
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/default_access <<EOF
{
	"master_key": "MASTER_KEY",
	"action": "schema:default_access",
	"type": "note",
	"default_access": [
		{"public": true, "level": "write"}
	]
}
EOF
*/
type SchemaDefaultAccessHandler struct {
	AccessKey     router.Processor `preprocessor:"accesskey"`
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	PluginReady   router.Processor `preprocessor:"plugin_ready"`
	preprocessors []router.Processor
}

type schemaDefaultAccessPayload struct {
	Type             string                   `mapstructure:"type"`
	RawDefaultAccess []map[string]interface{} `mapstructure:"default_access"`
	ACL              skydb.RecordACL
}

type schemaDefaultAccessResponse struct {
	Type          string                   `json:"type"`
	DefaultAccess []map[string]interface{} `json:"default_access,omitempty"`
}

func (h *SchemaDefaultAccessHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.AccessKey,
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
		h.PluginReady,
	}
}

func (h *SchemaDefaultAccessHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

func (payload *schemaDefaultAccessPayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}

	acl := skydb.RecordACL{}
	for _, v := range payload.RawDefaultAccess {
		ace := skydb.RecordACLEntry{}
		if err := (*skyconv.MapACLEntry)(&ace).FromMap(v); err != nil {
			return skyerr.NewInvalidArgument("invalid default_access entry", []string{"default_access"})
		}
		acl = append(acl, ace)
	}

	payload.ACL = acl

	return payload.Validate()
}

func (payload *schemaDefaultAccessPayload) Validate() skyerr.Error {
	if payload.Type == "" {
		return skyerr.NewInvalidArgument("missing required fields", []string{"type"})
	}

	return nil
}

func (h *SchemaDefaultAccessHandler) Handle(rpayload *router.Payload, response *router.Response) {
	payload := schemaDefaultAccessPayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	c := rpayload.Database.Conn()
	err := c.SetRecordDefaultAccess(payload.Type, payload.ACL)

	if err != nil {
		if skyErr, isSkyErr := err.(skyerr.Error); isSkyErr {
			response.Err = skyErr
		} else {
			response.Err = skyerr.MakeError(err)
		}
		return
	}

	response.Result = schemaDefaultAccessResponse{
		Type:          payload.Type,
		DefaultAccess: payload.RawDefaultAccess,
	}
}
