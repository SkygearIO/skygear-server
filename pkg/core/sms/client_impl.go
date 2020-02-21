package sms

import (
	"errors"

	"github.com/skygeario/skygear-server/pkg/core/config"
)

var ErrNoAvailableClient = errors.New("no available SMS client")

type ClientImpl struct {
	appConfig    *config.AppConfiguration
	nexmoClient  *NexmoClient
	twilioClient *TwilioClient
}

func NewClient(appConfig *config.AppConfiguration) *ClientImpl {
	nexmoConfig := appConfig.Nexmo
	twilioConfig := appConfig.Twilio

	var nexmoClient *NexmoClient
	if nexmoConfig.IsValid() {
		nexmoClient = NewNexmoClient(nexmoConfig)
	}

	var twilioClient *TwilioClient
	if twilioConfig.IsValid() {
		twilioClient = NewTwilioClient(twilioConfig)
	}

	return &ClientImpl{
		appConfig:    appConfig,
		nexmoClient:  nexmoClient,
		twilioClient: twilioClient,
	}
}

func (c *ClientImpl) Send(to string, body string) error {
	if c.nexmoClient != nil {
		return c.nexmoClient.Send(to, body)
	}
	if c.twilioClient != nil {
		return c.twilioClient.Send(to, body)
	}
	return ErrNoAvailableClient
}
