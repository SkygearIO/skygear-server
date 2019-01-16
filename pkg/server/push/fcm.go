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
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	"google.golang.org/api/option"
)

type fcmPushClient interface {
	Send(message interface{}) (interface{}, error)
}

type fcmHTTPPushClient struct {
	ctx                context.Context
	fcmMessagingClient *messaging.Client
}

func newFCMHTTPPushClient(serviceAccountKey string) (fcmPushClient, error) {
	ctx := context.Background()
	opt := option.WithCredentialsJSON([]byte(serviceAccountKey))
	app, err := firebase.NewApp(ctx, nil, opt)

	if err != nil {
		return nil, err
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &fcmHTTPPushClient{
		ctx:                ctx,
		fcmMessagingClient: client,
	}, nil
}

func (c *fcmHTTPPushClient) Send(message interface{}) (interface{}, error) {
	fcmMessage, _ := message.(*messaging.Message)
	return c.fcmMessagingClient.Send(c.ctx, fcmMessage)
}

var createFCMHTTPPushClient = newFCMHTTPPushClient
var isRegistrationTokenNotRegistered = messaging.IsRegistrationTokenNotRegistered

// FCMPusher sends push notifications via FCM.
type FCMPusher struct {
	conn              skydb.Conn
	ServiceAccountKey string
}

func NewFCMPusher(connOpener func() (skydb.Conn, error), serviceAccountKey string) (*FCMPusher, error) {
	conn, err := connOpener()
	if err != nil {
		return nil, err
	}
	return &FCMPusher{
		conn:              conn,
		ServiceAccountKey: serviceAccountKey,
	}, nil
}

// Send sends the dictionary represented by m to device.
func (p *FCMPusher) Send(m Mapper, device skydb.Device) error {
	if device.Token == "" {
		return fmt.Errorf("fcm: empty device token: %v", device.ID)
	}

	androidConfig := messaging.AndroidConfig{}
	if err := mapFCMMessage(m, &androidConfig); err != nil {
		log.Errorf("fcm: failed to convert fcm message: %v", err)
		return err
	}
	message := messaging.Message{
		Token:   device.Token,
		Android: &androidConfig,
	}

	client, err := createFCMHTTPPushClient(p.ServiceAccountKey)
	if err != nil {
		log.Errorf("fcm: failed to init fcm push client: %v", err)
		return err
	}

	response, err := client.Send(&message)
	if err != nil {
		log.Errorf("fcm: failed to call message api: %v", err)
		p.handleFailedNotification(device, err)
		return err
	}

	log.Info("fcm: push notification is sent: %v", response)

	return nil
}

func (p *FCMPusher) handleFailedNotification(device skydb.Device, err error) {
	if isRegistrationTokenNotRegistered(err) {
		log.WithFields(logrus.Fields{
			"deviceID":    device.ID,
			"deviceToken": device.Token,
			"error":       err,
		}).Info("fcm: delete device token")
		if err := p.conn.DeleteDevice(device.ID); err != nil && err != skydb.ErrDeviceNotFound {
			log.WithFields(logrus.Fields{
				"deviceID":    device.ID,
				"deviceToken": device.Token,
				"error":       err,
			}).Error("fcm: failed to delete device token")
			return
		}
	}
	return
}

func mapFCMMessage(mapper Mapper, msg interface{}) error {
	m := mapper.Map()
	var fcmMap map[string]interface{}
	fcmMap, ok := m["fcm"].(map[string]interface{})

	if !ok {
		fcmMap, _ = m["gcm"].(map[string]interface{})
	}

	if fcmMap == nil {
		return errors.New("fcm: payload has no fcm dictionary")
	}

	config := mapstructure.DecoderConfig{
		TagName: "json",
		Result:  msg,
	}
	// NewDecoder only returns error when DecoderConfig.Result
	// is not a pointer.
	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		panic(err)
	}

	err = decoder.Decode(fcmMap)

	return err
}
