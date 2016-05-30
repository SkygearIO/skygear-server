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

package skydbtest

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/skygeario/skygear-server/skydb"
)

// MapConn is a naive memory implementation of skydb.Conn
type MapConn struct {
	UserMap         map[string]skydb.UserInfo
	usernameMap     map[string]skydb.UserInfo
	emailMap        map[string]skydb.UserInfo
	recordAccessMap map[string]skydb.RecordACL
	skydb.Conn
}

// NewMapConn returns a new MapConn.
func NewMapConn() *MapConn {
	return &MapConn{
		UserMap:         map[string]skydb.UserInfo{},
		usernameMap:     map[string]skydb.UserInfo{},
		emailMap:        map[string]skydb.UserInfo{},
		recordAccessMap: map[string]skydb.RecordACL{},
	}
}

// CreateUser creates a UserInfo in UserMap.
func (conn *MapConn) CreateUser(userinfo *skydb.UserInfo) error {
	if _, existed := conn.UserMap[userinfo.ID]; existed {
		return skydb.ErrUserDuplicated
	}
	if _, existed := conn.usernameMap[userinfo.Username]; existed {
		return skydb.ErrUserDuplicated
	}
	if _, existed := conn.emailMap[userinfo.Email]; existed {
		return skydb.ErrUserDuplicated
	}

	conn.UserMap[userinfo.ID] = *userinfo
	conn.usernameMap[strings.ToLower(userinfo.Username)] = *userinfo
	conn.emailMap[strings.ToLower(userinfo.Email)] = *userinfo
	return nil
}

// GetUser returns a UserInfo in UserMap.
func (conn *MapConn) GetUser(id string, userinfo *skydb.UserInfo) error {
	u, ok := conn.UserMap[id]
	if !ok {
		return skydb.ErrUserNotFound
	}

	*userinfo = u
	return nil
}

// GetUserByUsernameEmail returns a UserInfo in UserMap by email address.
func (conn *MapConn) GetUserByUsernameEmail(username string, email string, userinfo *skydb.UserInfo) error {
	var (
		u  skydb.UserInfo
		ok bool
	)
	if email == "" {
		u, ok = conn.usernameMap[strings.ToLower(username)]
	} else if username == "" {
		u, ok = conn.emailMap[strings.ToLower(email)]
	} else {
		u, ok = conn.usernameMap[username]
		if u.Email != email {
			ok = false
		}
	}

	if !ok {
		return skydb.ErrUserNotFound
	}

	*userinfo = u
	return nil
}

// GetUserByPrincipalID returns a UserInfo by its principalID.
func (conn *MapConn) GetUserByPrincipalID(principalID string, userinfo *skydb.UserInfo) error {
	for _, u := range conn.UserMap {
		if _, ok := u.Auth[principalID]; ok {
			*userinfo = u
			return nil
		}
	}

	return skydb.ErrUserNotFound
}

// QueryUser is not implemented.
func (conn *MapConn) QueryUser(emails []string) ([]skydb.UserInfo, error) {
	panic("not implemented")
}

// UpdateUser updates an existing UserInfo in UserMap.
func (conn *MapConn) UpdateUser(userinfo *skydb.UserInfo) error {
	if _, ok := conn.UserMap[userinfo.ID]; !ok {
		return skydb.ErrUserNotFound
	}

	conn.UserMap[userinfo.ID] = *userinfo
	return nil
}

// DeleteUser remove an existing in UserMap.
func (conn *MapConn) DeleteUser(id string) error {
	if _, ok := conn.UserMap[id]; !ok {
		return skydb.ErrUserNotFound
	}

	delete(conn.UserMap, id)
	return nil
}

// GetAdminRoles is not implemented.
func (conn *MapConn) GetAdminRoles() ([]string, error) {
	return []string{
		"admin",
	}, nil
}

// SetAdminRoles is not implemented.
func (conn *MapConn) SetAdminRoles(roles []string) error {
	panic("not implemented")
}

// SetDefaultRoles is not implemented.
func (conn *MapConn) SetDefaultRoles(roles []string) error {
	panic("not implemented")
}

// SetRecordAccess sets record creation access
func (conn *MapConn) SetRecordAccess(recordType string, acl skydb.RecordACL) error {
	conn.recordAccessMap[recordType] = acl
	return nil
}

// GetRecordAccess returns record creation access of a specific type
func (conn *MapConn) GetRecordAccess(recordType string) (skydb.RecordACL, error) {
	acl, gotIt := conn.recordAccessMap[recordType]
	if !gotIt {
		acl = skydb.NewRecordACL([]skydb.RecordACLEntry{})
	}

	return acl, nil
}

// GetAsset is not implemented.
func (conn *MapConn) GetAsset(name string, asset *skydb.Asset) error {
	panic("not implemented")
}

// SaveAsset is not implemented.
func (conn *MapConn) SaveAsset(asset *skydb.Asset) error {
	panic("not implemented")
}

// QueryRelation is not implemented.
func (conn *MapConn) QueryRelation(user string, name string, direction string, config skydb.QueryConfig) []skydb.UserInfo {
	panic("not implemented")
}

// QueryRelationCount is not implemented.
func (conn *MapConn) QueryRelationCount(user string, name string, direction string) (uint64, error) {
	panic("not implemented")
}

// AddRelation is not implemented.
func (conn *MapConn) AddRelation(user string, name string, targetUser string) error {
	panic("not implemented")
}

// RemoveRelation is not implemented.
func (conn *MapConn) RemoveRelation(user string, name string, targetUser string) error {
	panic("not implemented")
}

// GetDevice is not implemented.
func (conn *MapConn) GetDevice(id string, device *skydb.Device) error {
	panic("not implemented")
}

// QueryDevicesByUser is not implemented.
func (conn *MapConn) QueryDevicesByUser(user string) ([]skydb.Device, error) {
	panic("not implemented")
}

// SaveDevice is not implemented.
func (conn *MapConn) SaveDevice(device *skydb.Device) error {
	panic("not implemented")
}

// DeleteDevice is not implemented.
func (conn *MapConn) DeleteDevice(id string) error {
	panic("not implemented")
}

// DeleteDeviceByToken is not implemented.
func (conn *MapConn) DeleteDeviceByToken(token string, t time.Time) error {
	panic("not implemented")
}

// DeleteEmptyDevicesByTime is not implemented.
func (conn *MapConn) DeleteEmptyDevicesByTime(t time.Time) error {
	panic("not implemented")
}

// PublicDB is not implemented.
func (conn *MapConn) PublicDB() skydb.Database {
	panic("not implemented")
}

// PrivateDB is not implemented.
func (conn *MapConn) PrivateDB(userKey string) skydb.Database {
	panic("not implemented")
}

// Subscribe is not implemented.
func (conn *MapConn) Subscribe(recordEventChan chan skydb.RecordEvent) error {
	panic("not implemented")
}

// Close does nothing.
func (conn *MapConn) Close() error {
	// do nothing
	return nil
}

// RecordMap is a string=>Record map
type RecordMap map[string]skydb.Record

// SubscriptionMap is a string=>Subscription map
type SubscriptionMap map[string]skydb.Subscription

// RecordSchemaMap is a string=>RecordSchema map
type RecordSchemaMap map[string]skydb.RecordSchema

//recordType string, acl RecordACL

// MapDB is a naive memory implementation of skydb.Database.
type MapDB struct {
	RecordMap       RecordMap
	SubscriptionMap SubscriptionMap
	RecordSchemaMap RecordSchemaMap
	DBConn          skydb.Conn
	skydb.Database
}

// NewMapDB returns a new MapDB ready for use.
func NewMapDB() *MapDB {
	return &MapDB{
		RecordMap:       RecordMap{},
		SubscriptionMap: SubscriptionMap{},
		RecordSchemaMap: RecordSchemaMap{},
		DBConn:          &MapConn{},
	}
}

// ID returns a mock Database ID.
func (db *MapDB) ID() string {
	return "map-db"
}

func (db *MapDB) UserRecordType() string {
	return "user"
}

// Get returns a Record from RecordMap.
func (db *MapDB) Get(id skydb.RecordID, record *skydb.Record) error {
	r, ok := db.RecordMap[id.String()]
	if !ok {
		return skydb.ErrRecordNotFound
	}
	*record = r
	return nil

}

// Save assigns Record to RecordMap.
func (db *MapDB) Save(record *skydb.Record) error {
	db.RecordMap[record.ID.String()] = *record
	return nil
}

// Delete remove the specified key from RecordMap.
func (db *MapDB) Delete(id skydb.RecordID) error {
	_, ok := db.RecordMap[id.String()]
	if !ok {
		return skydb.ErrRecordNotFound
	}
	delete(db.RecordMap, id.String())
	return nil
}

// Query is not implemented.
func (db *MapDB) Query(query *skydb.Query) (*skydb.Rows, error) {
	panic("skydbtest: MapDB.Query not supported")
}

// Extend store the type of the field.
func (db *MapDB) Extend(recordType string, schema skydb.RecordSchema) error {
	if _, ok := db.RecordSchemaMap[recordType]; ok {
		for fieldName, fieldType := range schema {
			if _, ok := db.RecordSchemaMap[recordType][fieldName]; ok {
				ft := db.RecordSchemaMap[recordType][fieldName]
				if !reflect.DeepEqual(ft, fieldType) {
					return fmt.Errorf("Wrong type")
				}
			}
			db.RecordSchemaMap[recordType][fieldName] = fieldType
		}
	} else {
		db.RecordSchemaMap[recordType] = schema
	}
	return nil
}

func (db *MapDB) RenameSchema(recordType, oldColumnName, newColumnName string) error {
	if _, ok := db.RecordSchemaMap[recordType]; !ok {
		return fmt.Errorf("record type %s does not exist", recordType)
	}
	if _, ok := db.RecordSchemaMap[recordType][oldColumnName]; !ok {
		return fmt.Errorf("column %s does not exist", oldColumnName)
	}
	if _, ok := db.RecordSchemaMap[recordType][newColumnName]; ok {
		if !reflect.DeepEqual(
			db.RecordSchemaMap[recordType][oldColumnName],
			db.RecordSchemaMap[recordType][newColumnName],
		) {
			return fmt.Errorf("column type conflict")
		}
	}
	db.RecordSchemaMap[recordType][newColumnName] = db.RecordSchemaMap[recordType][oldColumnName]
	delete(db.RecordSchemaMap[recordType], oldColumnName)

	return nil
}

func (db *MapDB) DeleteSchema(recordType, columnName string) error {
	if _, ok := db.RecordSchemaMap[recordType]; !ok {
		return fmt.Errorf("record type %s does not exist", recordType)
	}
	if _, ok := db.RecordSchemaMap[recordType][columnName]; !ok {
		return fmt.Errorf("column %s does not exist", columnName)
	}
	delete(db.RecordSchemaMap[recordType], columnName)
	return nil
}

// GetSchema returns the record schema of a record type
func (db *MapDB) GetSchema(recordType string) (skydb.RecordSchema, error) {
	if _, ok := db.RecordSchemaMap[recordType]; !ok {
		return nil, fmt.Errorf("record type %s does not exist", recordType)
	}
	return db.RecordSchemaMap[recordType], nil
}

// GetRecordSchemas returns a list of all existing record type
func (db *MapDB) GetRecordSchemas() (map[string]skydb.RecordSchema, error) {
	return db.RecordSchemaMap, nil
}

// GetSubscription return a Subscription from SubscriptionMap.
func (db *MapDB) GetSubscription(name string, deviceID string, subscription *skydb.Subscription) error {
	s, ok := db.SubscriptionMap[deviceID+"/"+name]
	if !ok {
		return skydb.ErrSubscriptionNotFound
	}
	*subscription = s
	return nil
}

// SaveSubscription assigns to SubscriptionMap.
func (db *MapDB) SaveSubscription(subscription *skydb.Subscription) error {
	db.SubscriptionMap[subscription.DeviceID+"/"+subscription.ID] = *subscription
	return nil
}

// DeleteSubscription deletes the specified key from SubscriptionMap.
func (db *MapDB) DeleteSubscription(name string, deviceID string) error {
	key := deviceID + "/" + name
	_, ok := db.SubscriptionMap[key]
	if !ok {
		return skydb.ErrSubscriptionNotFound
	}
	delete(db.SubscriptionMap, key)
	return nil
}

// MockTxDatabase implements and records TxDatabase's methods and delegates other
// calls to underlying Database
type MockTxDatabase struct {
	DidBegin, DidCommit, DidRollback bool
	skydb.Database
}

func NewMockTxDatabase(backingDB skydb.Database) *MockTxDatabase {
	return &MockTxDatabase{Database: backingDB}
}

func (db *MockTxDatabase) Begin() error {
	db.DidBegin = true
	return nil
}

func (db *MockTxDatabase) Commit() error {
	db.DidCommit = true
	return nil
}

func (db *MockTxDatabase) Rollback() error {
	db.DidRollback = true
	return nil
}

var _ skydb.TxDatabase = &MockTxDatabase{}

var (
	_ skydb.Conn       = NewMapConn()
	_ skydb.Database   = NewMapDB()
	_ skydb.TxDatabase = &MockTxDatabase{}
)
