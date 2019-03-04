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

package push

import (
	"testing"

	"github.com/skygeario/skygear-server/pkg/server/skydb"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/maddevsio/fcm.v1"
)

type mockFCMLegacyHTTPPushClient struct {
	lastMessage interface{}
}

func (c *mockFCMLegacyHTTPPushClient) Send(message interface{}) (interface{}, error) {
	c.lastMessage = message
	return nil, nil
}

func TestLegacyFCMSend(t *testing.T) {
	Convey("LegacyFCMPusher", t, func() {
		mockClient := &mockFCMHTTPPushClient{}
		createFCMLegacyHTTPPushClient = func(serverKey string) (fcmPushClient, error) {
			return mockClient, nil
		}
		defer func() {
			createFCMLegacyHTTPPushClient = newFCMLegacyHTTPPushClient
		}()

		pusher := LegacyFCMPusher{
			APIKey: "fakeAPIKey",
		}
		device := skydb.Device{
			Token: "deviceToken",
		}

		Convey("sends notification", func() {
			err := pusher.Send(MapMapper{
				"fcm": map[string]interface{}{
					"notification": map[string]interface{}{
						"title": "You have got a message",
						"body":  "This is a message.",
						"icon":  "myicon",
						"sound": "default",
					},
					"data": map[string]interface{}{
						"string":  "value",
						"string2": "value2",
					},
				},
			}, device)

			So(err, ShouldBeNil)
			So(mockClient.lastMessage, ShouldResemble, fcm.Message{
				RegistrationIDs: []string{"deviceToken"},
				Notification: fcm.Notification{
					Title: "You have got a message",
					Body:  "This is a message.",
					Icon:  "myicon",
					Sound: "default",
				},
				Data: map[string]interface{}{
					"string":  "value",
					"string2": "value2",
				},
			})
		})

		Convey("should use gcm key as fallback in payload", func() {
			err := pusher.Send(MapMapper{
				"gcm": map[string]interface{}{
					"notification": map[string]interface{}{
						"title": "You have got a message",
						"body":  "This is a message.",
						"icon":  "myicon",
						"sound": "default",
					},
					"data": map[string]interface{}{
						"string":  "value",
						"string2": "value2",
					},
				},
			}, device)

			So(err, ShouldBeNil)
			So(mockClient.lastMessage, ShouldResemble, fcm.Message{
				RegistrationIDs: []string{"deviceToken"},
				Notification: fcm.Notification{
					Title: "You have got a message",
					Body:  "This is a message.",
					Icon:  "myicon",
					Sound: "default",
				},
				Data: map[string]interface{}{
					"string":  "value",
					"string2": "value2",
				},
			})
		})

	})

}
