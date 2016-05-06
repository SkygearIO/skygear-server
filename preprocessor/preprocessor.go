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

package preprocessor

import (
	"net/http"

	log "github.com/Sirupsen/logrus"

	"github.com/skygeario/skygear-server/router"
	"github.com/skygeario/skygear-server/skydb"
	"github.com/skygeario/skygear-server/skyerr"
)

type InjectUserIfPresent struct {
}

func (p InjectUserIfPresent) Preprocess(payload *router.Payload, response *router.Response) int {
	if payload.UserInfoID == "" {
		log.Debugln("injectUser: empty UserInfoID, skipping")
		return http.StatusOK
	}

	conn := payload.DBConn
	userinfo := skydb.UserInfo{}
	if err := conn.GetUser(payload.UserInfoID, &userinfo); err != nil {
		log.Errorf("Cannot find UserInfo.ID = %#v\n", payload.UserInfoID)
		response.Err = skyerr.NewError(skyerr.UnexpectedUserInfoNotFound, err.Error())
		return http.StatusInternalServerError
	}

	if payload.AccessKey == router.MasterAccessKey {
		userinfo.WithMasterKey = true
	}

	payload.UserInfo = &userinfo

	return http.StatusOK
}

type InjectDatabase struct {
}

func (p InjectDatabase) Preprocess(payload *router.Payload, response *router.Response) int {
	conn := payload.DBConn

	databaseID, _ := payload.Data["database_id"].(string)
	switch databaseID {
	case "_private":
		if payload.UserInfo != nil {
			payload.Database = conn.PrivateDB(payload.UserInfo.ID)
		} else {
			response.Err = skyerr.NewError(skyerr.NotAuthenticated, "Authentication is needed for private DB access")
			return http.StatusUnauthorized
		}
	case "_public":
		payload.Database = conn.PublicDB()
	default:
		payload.Database = conn.PublicDB()
	}

	return http.StatusOK
}

type InjectPublicDatabase struct {
}

func (p InjectPublicDatabase) Preprocess(payload *router.Payload, response *router.Response) int {
	conn := payload.DBConn
	payload.Database = conn.PublicDB()
	return http.StatusOK
}

type RequireUserForWrite struct {
}

func (p RequireUserForWrite) Preprocess(payload *router.Payload, response *router.Response) int {
	if payload.UserInfo == nil {
		response.Err = skyerr.NewError(skyerr.PermissionDenied, "write is not allowed")
		return http.StatusUnauthorized
	}

	return http.StatusOK
}
