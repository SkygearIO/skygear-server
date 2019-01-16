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

	"firebase.google.com/go/messaging"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	. "github.com/smartystreets/goconvey/convey"
)

type mockFCMHTTPPushClient struct {
	lastMessage interface{}
}

func (c *mockFCMHTTPPushClient) Send(message interface{}) (interface{}, error) {
	fcmMessage, ok := message.(*messaging.Message)
	if ok {
		switch fcmMessage.Token {
		case "not-registered-token":
			return nil, &mockFCMError{
				Code: "registration-token-not-registered",
			}
		case "invalid-token":
			return nil, &mockFCMError{
				Code: "invalid-registration-token",
			}
		}
	}
	c.lastMessage = message
	return nil, nil
}

type mockFCMError struct {
	Code   string
	String string
}

func (me *mockFCMError) Error() string {
	return me.String
}

func TestFCMSend(t *testing.T) {
	Convey("FCMPusher", t, func() {
		conn := &mockConn{}
		mockClient := &mockFCMHTTPPushClient{}

		// mock fcm http push client
		createFCMHTTPPushClient = func(serviceAccountKey string) (fcmPushClient, error) {
			return mockClient, nil
		}
		defer func() {
			createFCMHTTPPushClient = newFCMHTTPPushClient
		}()

		// mock isRegistrationTokenNotRegistered checking
		isRegistrationTokenNotRegistered = func(err error) bool {
			mockErr, ok := err.(*mockFCMError)
			if ok && mockErr.Code == "registration-token-not-registered" {
				return true
			}
			return false
		}
		defer func() {
			isRegistrationTokenNotRegistered = messaging.IsRegistrationTokenNotRegistered
		}()

		pusher := FCMPusher{
			conn:              conn,
			ServiceAccountKey: "fakeServiceAccount",
		}
		device := skydb.Device{
			Token: "device-token",
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
			So(mockClient.lastMessage, ShouldResemble, &messaging.Message{
				Token: "device-token",
				Android: &messaging.AndroidConfig{
					Notification: &messaging.AndroidNotification{
						Title: "You have got a message",
						Body:  "This is a message.",
						Icon:  "myicon",
						Sound: "default",
					},
					Data: map[string]string{
						"string":  "value",
						"string2": "value2",
					},
				},
			})
		})

		Convey("only accept string value in data", func() {
			err := pusher.Send(MapMapper{
				"fcm": map[string]interface{}{
					"notification": map[string]interface{}{
						"title": "You have got a message",
						"body":  "This is a message.",
						"icon":  "myicon",
						"sound": "default",
						"badge": "5",
					},
					"data": map[string]interface{}{
						"string":  "value",
						"integer": 1,
					},
				},
			}, device)

			So(err, ShouldNotBeNil)
		})

		Convey("should delete device if device is not registered", func() {
			notRegisteredDevice := skydb.Device{
				ID:    "not-registered-token-id",
				Token: "not-registered-token",
			}

			err := pusher.Send(MapMapper{
				"fcm": map[string]interface{}{
					"notification": map[string]interface{}{
						"title": "You have got a message",
						"body":  "This is a message.",
						"icon":  "myicon",
						"sound": "default",
						"badge": "5",
					},
					"data": map[string]interface{}{
						"string": "value",
					},
				},
			}, notRegisteredDevice)

			So(err, ShouldNotBeNil)
			So(len(conn.calls), ShouldEqual, 1)
			So(conn.calls[0].id, ShouldEqual, "not-registered-token-id")
		})

		Convey("should not delete device with other error", func() {
			invalidDevice := skydb.Device{
				ID:    "invalid-token-id",
				Token: "invalid-token",
			}

			err := pusher.Send(MapMapper{
				"fcm": map[string]interface{}{
					"notification": map[string]interface{}{
						"title": "You have got a message",
						"body":  "This is a message.",
						"icon":  "myicon",
						"sound": "default",
						"badge": "5",
					},
					"data": map[string]interface{}{
						"string": "value",
					},
				},
			}, invalidDevice)

			So(err, ShouldNotBeNil)
			So(len(conn.calls), ShouldEqual, 0)
		})
	})

}
