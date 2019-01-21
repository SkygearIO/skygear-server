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
	"fmt"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	"gopkg.in/maddevsio/fcm.v1"
)

type fcmLegacyHTTPPushClient struct {
	fcmMessagingClient *fcm.FCM
}

func newFCMLegacyHTTPPushClient(apiKey string) (fcmPushClient, error) {
	return &fcmLegacyHTTPPushClient{
		fcmMessagingClient: fcm.NewFCM(apiKey),
	}, nil
}

func (c *fcmLegacyHTTPPushClient) Send(message interface{}) (response interface{}, err error) {
	fcmMessage, _ := message.(fcm.Message)
	return c.fcmMessagingClient.Send(fcmMessage)
}

var createFCMLegacyHTTPPushClient = newFCMLegacyHTTPPushClient

// LegacyFCMPusher sends push notifications via GCM.
type LegacyFCMPusher struct {
	APIKey string
}

// Send sends the dictionary represented by m to device.
func (p *LegacyFCMPusher) Send(m Mapper, device skydb.Device) error {
	if device.Token == "" {
		return fmt.Errorf("fcm: empty device token: %v", device.ID)
	}

	message := fcm.Message{}

	if err := mapFCMMessage(m, &message); err != nil {
		log.Errorf("fcm/key: failed to convert fcm message: %v", err)
		return err
	}

	message.RegistrationIDs = []string{device.Token}

	c, _ := createFCMLegacyHTTPPushClient(p.APIKey)
	if _, err := c.Send(message); err != nil {
		log.Errorf("fcm/key: failed to send fcm Notification: %v", err)
		return err
	}

	return nil
}
