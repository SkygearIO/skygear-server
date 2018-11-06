package userprofile

import (
	"github.com/franela/goreq"
	"github.com/sirupsen/logrus"
)

type recordStoreImpl struct {
	storeURL string
	apiKey   string
	logger   *logrus.Entry
}

// NewUserProfileRecordStore returns a record-gear based user profile store implementation
func NewUserProfileRecordStore(storeURL string, apiKey string, logger *logrus.Entry) Store {
	return &recordStoreImpl{
		storeURL: storeURL,
		apiKey:   apiKey,
		logger:   logger,
	}
}

func (u *recordStoreImpl) CreateUserProfile(userID string, accessToken string, data Data) (profile UserProfile, err error) {
	payload := make(Record)

	payload["_id"] = "user/" + userID
	for k, v := range data {
		payload[k] = v
	}

	var records []Record
	records = append(records, payload)

	body := make(map[string][]Record)
	body["records"] = records

	resp, err := goreq.Request{
		Method: "POST",
		Uri:    u.storeURL + "save",
		Body:   body,
	}.
		WithHeader("X-Skygear-Api-Key", u.apiKey).
		WithHeader("X-Skygear-Access-Token", accessToken).
		Do()

	if err != nil {
		return
	}

	err = resp.Body.FromJsonTo(&profile)
	if err != nil {
		return
	}

	return
}

func (u *recordStoreImpl) GetUserProfile(userID string, accessToken string) (profile UserProfile, err error) {
	var ids []string
	ids = append(ids, "user/"+userID)

	body := make(map[string][]string)
	body["ids"] = ids

	resp, err := goreq.Request{
		Method: "POST",
		Uri:    u.storeURL + "fetch",
		Body:   body,
	}.
		WithHeader("X-Skygear-Api-Key", u.apiKey).
		WithHeader("X-Skygear-Access-Token", accessToken).
		Do()

	if err != nil {
		return
	}

	err = resp.Body.FromJsonTo(&profile)
	if err != nil {
		return
	}

	return
}

func (u *recordStoreImpl) CanWithInTx() (withIn bool) {
	// gear to gear communication should commit tx first
	withIn = false
	return
}
