package userprofile

import (
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"time"
)

type MockUserProfileStoreImpl struct {
	Data        map[string]map[string]interface{}
	TimeNowfunc MockTimeNowfunc
}

type MockTimeNowfunc func() time.Time

func NewMockUserProfileStore() *MockUserProfileStoreImpl {
	return &MockUserProfileStoreImpl{
		Data:        map[string]map[string]interface{}{},
		TimeNowfunc: func() time.Time { return time.Time{} },
	}
}

func (u MockUserProfileStoreImpl) CreateUserProfile(userID string, authInfo *authinfo.AuthInfo, data Data) (profile UserProfile, err error) {
	u.Data[userID] = make(map[string]interface{})
	u.Data[userID] = data
	now := u.TimeNowfunc()
	profile = UserProfile{
		Meta: Meta{
			ID:         "user/" + userID,
			Type:       "record",
			RecordID:   userID,
			RecordType: "user",
			Access:     nil,
			OwnerID:    userID,
			CreatedAt:  now,
			CreatedBy:  userID,
			UpdatedAt:  now,
			UpdatedBy:  userID,
		},
		Data: data,
	}
	return
}

func (u MockUserProfileStoreImpl) GetUserProfile(userID string, accessToken string) (profile UserProfile, err error) {
	data := u.Data[userID]
	now := u.TimeNowfunc()
	profile = UserProfile{
		Meta: Meta{
			ID:         "user/" + userID,
			Type:       "record",
			RecordID:   userID,
			RecordType: "user",
			Access:     nil,
			OwnerID:    userID,
			CreatedAt:  now,
			CreatedBy:  userID,
			UpdatedAt:  now,
			UpdatedBy:  userID,
		},
		Data: data,
	}
	return
}

func (u MockUserProfileStoreImpl) UpdateUserProfile(userID string, authInfo *authinfo.AuthInfo, data Data) (profile UserProfile, err error) {
	newData := u.Data[userID]
	for k := range data {
		newData[k] = data[k]
	}

	u.Data[userID] = newData
	now := u.TimeNowfunc()
	profile = UserProfile{
		Meta: Meta{
			ID:         "user/" + userID,
			Type:       "record",
			RecordID:   userID,
			RecordType: "user",
			Access:     nil,
			OwnerID:    userID,
			CreatedAt:  now,
			CreatedBy:  userID,
			UpdatedAt:  now,
			UpdatedBy:  userID,
		},
		Data: newData,
	}
	return
}
