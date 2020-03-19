package model

import (
	"fmt"
	"regexp"

	"github.com/ua-parser/uap-go/uaparser"
)

var uaParser = uaparser.NewFromSaved()
var skygearUARegex = regexp.MustCompile(`^(.*)/(\d+)(?:\.(\d+)|)(?:\.(\d+)|)(?:\.(\d+)|) \(Skygear;`)

type UserAgent struct {
	Raw         string `json:"raw"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	OS          string `json:"os"`
	OSVersion   string `json:"os_version"`
	DeviceName  string `json:"device_name"`
	DeviceModel string `json:"device_model"`
}

// @JSONSchema
const UserAgentSchema = `
{
	"$id": "#UserAgent",
	"type": "object",
	"properties": {
		"raw": { "type": "string" },
		"name": { "type": "string" },
		"version": { "type": "string" },
		"os": { "type": "string" },
		"os_version": { "type": "string" },
		"device_name": { "type": "string" },
		"device_model": { "type": "string" }
	}
}
`

func ParseUserAgent(ua string) (mUA UserAgent) {
	mUA.Raw = ua

	client := uaParser.Parse(ua)
	if matches := skygearUARegex.FindStringSubmatch(ua); len(matches) > 0 {
		client.UserAgent.Family = matches[1]
		client.UserAgent.Major = matches[2]
		client.UserAgent.Minor = matches[3]
		client.UserAgent.Patch = matches[4]
	}

	if client.UserAgent.Family != "Other" {
		mUA.Name = client.UserAgent.Family
		mUA.Version = client.UserAgent.ToVersionString()
	}
	if client.Device.Family != "Other" {
		mUA.DeviceModel = fmt.Sprintf("%s %s", client.Device.Brand, client.Device.Model)
	}
	if client.Os.Family != "Other" {
		mUA.OS = client.Os.Family
		mUA.OSVersion = client.Os.ToVersionString()
	}

	return mUA
}
