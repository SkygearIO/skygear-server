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

package pq

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/skygeario/skygear-server/pkg/server/skydb"
	. "github.com/skygeario/skygear-server/pkg/server/skytest"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGet(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PrivateDB("getuser")
		So(db.Extend("record", skydb.RecordSchema{
			"string":   skydb.FieldType{Type: skydb.TypeString},
			"number":   skydb.FieldType{Type: skydb.TypeNumber},
			"datetime": skydb.FieldType{Type: skydb.TypeDateTime},
			"boolean":  skydb.FieldType{Type: skydb.TypeBoolean},
		}), ShouldBeNil)

		insertRow(t, c.Db(), `INSERT INTO app_com_oursky_skygear."record" `+
			`(_database_id, _id, _owner_id, _created_at, _created_by, _updated_at, _updated_by, "string", "number", "datetime", "boolean") `+
			`VALUES ('getuser', 'id0', 'getuser', '1988-02-06', 'getuser', '1988-02-06', 'getuser', 'string', 1, '1988-02-06', TRUE)`)
		insertRow(t, c.Db(), `INSERT INTO app_com_oursky_skygear."record" `+
			`(_database_id, _id, _owner_id, _created_at, _created_by, _updated_at, _updated_by, "string", "number", "datetime", "boolean") `+
			`VALUES ('getuser', 'id1', 'getuser', '1988-02-06', 'getuser', '1988-02-06', 'getuser', 'string', 1, '1988-02-06', TRUE)`)

		Convey("gets an existing record from database", func() {
			record := skydb.Record{}
			err := db.Get(skydb.NewRecordID("record", "id1"), &record)
			So(err, ShouldBeNil)

			So(record.ID, ShouldResemble, skydb.NewRecordID("record", "id1"))
			So(record.DatabaseID, ShouldResemble, "getuser")
			So(record.OwnerID, ShouldResemble, "getuser")
			So(record.CreatorID, ShouldResemble, "getuser")
			So(record.UpdaterID, ShouldResemble, "getuser")
			So(record.Data["string"], ShouldEqual, "string")
			So(record.Data["number"], ShouldEqual, 1)
			So(record.Data["boolean"], ShouldEqual, true)

			So(record.CreatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
			So(record.UpdatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
			So(record.Data["datetime"].(time.Time), ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
		})

		Convey("errors if gets a non-existing record", func() {
			record := skydb.Record{}
			err := db.Get(skydb.NewRecordID("record", "notexistid"), &record)
			So(err, ShouldEqual, skydb.ErrRecordNotFound)
		})
	})
}

func TestGetByIDs(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PrivateDB("getuser")
		So(db.Extend("record", skydb.RecordSchema{
			"string": skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)

		insertRow(t, c.Db(), `INSERT INTO app_com_oursky_skygear."record" `+
			`(_database_id, _id, _owner_id, _created_at, _created_by, _updated_at, _updated_by, "string") `+
			`VALUES ('getuser', 'id0', 'getuser', '1988-02-06', 'getuser', '1988-02-06', 'getuser', 'string')`)
		insertRow(t, c.Db(), `INSERT INTO app_com_oursky_skygear."record" `+
			`(_database_id, _id, _owner_id, _created_at, _created_by, _updated_at, _updated_by, "string") `+
			`VALUES ('getuser', 'id1', 'getuser', '1988-02-06', 'getuser', '1988-02-06', 'getuser', 'string')`)

		Convey("get one record", func() {
			scanner, err := db.GetByIDs([]skydb.RecordID{skydb.NewRecordID("record", "id1")})
			So(err, ShouldBeNil)

			scanner.Scan()
			record := scanner.Record()

			So(record.ID, ShouldResemble, skydb.NewRecordID("record", "id1"))
			So(record.DatabaseID, ShouldResemble, "getuser")
			So(record.OwnerID, ShouldResemble, "getuser")
			So(record.CreatorID, ShouldResemble, "getuser")
			So(record.UpdaterID, ShouldResemble, "getuser")
			So(record.Data["string"], ShouldEqual, "string")

			So(record.CreatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
			So(record.UpdatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
			noMore := scanner.Scan()
			So(noMore, ShouldEqual, false)
		})

		Convey("get one record with duplicated record ID", func() {
			scanner, err := db.GetByIDs([]skydb.RecordID{
				skydb.NewRecordID("record", "id1"),
				skydb.NewRecordID("record", "id1"),
			})
			So(err, ShouldBeNil)

			scanner.Scan()
			record := scanner.Record()

			So(record.ID, ShouldResemble, skydb.NewRecordID("record", "id1"))
			So(record.DatabaseID, ShouldResemble, "getuser")
			So(record.OwnerID, ShouldResemble, "getuser")
			So(record.CreatorID, ShouldResemble, "getuser")
			So(record.UpdaterID, ShouldResemble, "getuser")
			So(record.Data["string"], ShouldEqual, "string")

			So(record.CreatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
			So(record.UpdatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))

			noMore := scanner.Scan()
			So(noMore, ShouldEqual, false)
		})

		Convey("get record with one of them is placeholder", func() {
			scanner, err := db.GetByIDs([]skydb.RecordID{
				skydb.RecordID{},
				skydb.NewRecordID("record", "id1"),
			})
			So(err, ShouldBeNil)

			scanner.Scan()
			record := scanner.Record()

			So(record.ID, ShouldResemble, skydb.NewRecordID("record", "id1"))
			So(record.DatabaseID, ShouldResemble, "getuser")
			So(record.OwnerID, ShouldResemble, "getuser")
			So(record.CreatorID, ShouldResemble, "getuser")
			So(record.UpdaterID, ShouldResemble, "getuser")
			So(record.Data["string"], ShouldEqual, "string")

			So(record.CreatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))
			So(record.UpdatedAt, ShouldResemble, time.Date(1988, 2, 6, 0, 0, 0, 0, time.UTC))

			noMore := scanner.Scan()
			So(noMore, ShouldEqual, false)
		})

		Convey("get multiple record", func() {
			scanner, err := db.GetByIDs([]skydb.RecordID{
				skydb.NewRecordID("record", "id0"),
				skydb.NewRecordID("record", "id1"),
			})
			So(err, ShouldBeNil)

			scanner.Scan()
			record := scanner.Record()

			scanner.Scan()
			record2 := scanner.Record()
			So([]skydb.RecordID{
				record.ID,
				record2.ID,
			}, ShouldResemble, []skydb.RecordID{
				skydb.NewRecordID("record", "id0"),
				skydb.NewRecordID("record", "id1"),
			})

			noMore := scanner.Scan()
			So(noMore, ShouldEqual, false)
		})

	})
}

func TestSave(t *testing.T) {
	var c *conn
	Convey("Database", t, func() {
		c = getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PublicDB()
		So(db.Extend("note", skydb.RecordSchema{
			"content":   skydb.FieldType{Type: skydb.TypeString},
			"number":    skydb.FieldType{Type: skydb.TypeNumber},
			"timestamp": skydb.FieldType{Type: skydb.TypeDateTime},
		}), ShouldBeNil)

		record := skydb.Record{
			ID:        skydb.NewRecordID("note", "someid"),
			OwnerID:   "user_id",
			CreatedAt: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			CreatorID: "creator",
			UpdatedAt: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			UpdaterID: "updater",
			Data: map[string]interface{}{
				"content":   "some content",
				"number":    float64(1),
				"timestamp": time.Date(1988, 2, 6, 1, 1, 1, 1, time.UTC),
			},
		}

		Convey("creates record if it doesn't exist", func() {
			err := db.Save(&record)
			So(err, ShouldBeNil)
			So(record.DatabaseID, ShouldEqual, "")

			var (
				content   string
				number    float64
				timestamp time.Time
				ownerID   string
			)
			err = c.QueryRowx(
				"SELECT content, number, timestamp, _owner_id "+
					"FROM app_com_oursky_skygear.note WHERE _id = 'someid' and _database_id = ''").
				Scan(&content, &number, &timestamp, &ownerID)
			So(err, ShouldBeNil)
			So(content, ShouldEqual, "some content")
			So(number, ShouldEqual, float64(1))
			So(timestamp.In(time.UTC), ShouldResemble, time.Date(1988, 2, 6, 1, 1, 1, 0, time.UTC))
			So(ownerID, ShouldEqual, "user_id")
		})

		Convey("updates record if it already exists", func() {
			err := db.Save(&record)
			So(err, ShouldBeNil)
			So(record.DatabaseID, ShouldEqual, "")

			record.Set("content", "more content")
			err = db.Save(&record)
			So(err, ShouldBeNil)

			var content string
			err = c.QueryRowx("SELECT content FROM app_com_oursky_skygear.note WHERE _id = 'someid' and _database_id = ''").
				Scan(&content)
			So(err, ShouldBeNil)
			So(content, ShouldEqual, "more content")
		})

		Convey("error if saving with recordid already taken by other user", func() {
			ownerDB := c.PrivateDB("ownerid")
			err := ownerDB.Save(&record)
			So(err, ShouldBeNil)
			otherDB := c.PrivateDB("otheruserid")
			err = otherDB.Save(&record)
			// FIXME: Wrap me with skydb.ErrXXX
			So(err, ShouldNotBeNil)
		})

		Convey("ignore Record.DatabaseID when saving", func() {
			record.DatabaseID = "someuserid"
			err := db.Save(&record)
			So(err, ShouldBeNil)
			So(record.DatabaseID, ShouldEqual, "")

			var count int
			err = c.QueryRowx("SELECT count(*) FROM app_com_oursky_skygear.note WHERE _id = 'someid' and _database_id = 'someuserid'").
				Scan(&count)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)
		})

		Convey("REGRESSION: update record with attribute having capital letters", func() {
			So(db.Extend("note", skydb.RecordSchema{
				"noteOrder": skydb.FieldType{Type: skydb.TypeNumber},
			}), ShouldBeNil)

			record = skydb.Record{
				ID:      skydb.NewRecordID("note", "1"),
				OwnerID: "user_id",
				Data: map[string]interface{}{
					"noteOrder": 1,
				},
			}

			ShouldBeNil(db.Save(&record))

			record.Data["noteOrder"] = 2
			ShouldBeNil(db.Save(&record))

			var noteOrder int
			err := c.QueryRowx(`SELECT "noteOrder" FROM app_com_oursky_skygear.note WHERE _id = '1' and _database_id = ''`).
				Scan(&noteOrder)
			So(err, ShouldBeNil)
			So(noteOrder, ShouldEqual, 2)
		})

		Convey("errors if OwnerID not set", func() {
			record.OwnerID = ""
			err := db.Save(&record)
			So(err.Error(), ShouldEndWith, "got empty OwnerID")
		})

		Convey("ignore OwnerID when update", func() {
			err := db.Save(&record)
			So(err, ShouldBeNil)

			record.OwnerID = "user_id2"
			So(err, ShouldBeNil)

			var ownerID string
			err = c.QueryRowx(`SELECT "_owner_id" FROM app_com_oursky_skygear.note WHERE _id = 'someid' and _database_id = ''`).
				Scan(&ownerID)
			So(ownerID, ShouldEqual, "user_id")
		})
	})
}

func TestDelete(t *testing.T) {
	var c *conn
	Convey("Database", t, func() {
		c = getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PrivateDB("userid")

		So(db.Extend("note", skydb.RecordSchema{
			"content": skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)

		record := skydb.Record{
			ID:      skydb.NewRecordID("note", "someid"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"content": "some content",
			},
		}

		Convey("deletes existing record", func() {
			err := db.Save(&record)
			So(err, ShouldBeNil)

			err = db.Delete(skydb.NewRecordID("note", "someid"))
			So(err, ShouldBeNil)

			err = db.(*database).c.QueryRowx("SELECT * FROM app_com_oursky_skygear.note WHERE _id = 'someid' AND _database_id = 'userid'").Scan((*string)(nil))
			So(err, ShouldEqual, sql.ErrNoRows)
		})

		Convey("returns ErrRecordNotFound when record to delete doesn't exist", func() {
			err := db.Delete(skydb.NewRecordID("note", "notexistid"))
			So(err, ShouldEqual, skydb.ErrRecordNotFound)
		})

		Convey("return ErrRecordNotFound when deleting other user record", func() {
			err := db.Save(&record)
			So(err, ShouldBeNil)
			otherDB := c.PrivateDB("otheruserid")
			err = otherDB.Delete(skydb.NewRecordID("note", "someid"))
			So(err, ShouldEqual, skydb.ErrRecordNotFound)
		})
	})
}

func TestQuery(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		// fixture
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id1"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(1),
				"content":   "Hello World",
				"emotion":   nil,
			},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id2"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(2),
				"content":   "Bye World",
				"emotion":   nil,
			},
		}
		record3 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id3"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(3),
				"content":   "Good Hello",
				"emotion":   "happy",
			},
		}

		db := c.PrivateDB("userid")
		So(db.Extend("note", skydb.RecordSchema{
			"noteOrder": skydb.FieldType{Type: skydb.TypeNumber},
			"content":   skydb.FieldType{Type: skydb.TypeString},
			"emotion":   skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)

		err := db.Save(&record2)
		So(err, ShouldBeNil)
		err = db.Save(&record1)
		So(err, ShouldBeNil)
		err = db.Save(&record3)
		So(err, ShouldBeNil)

		Convey("queries records", func() {
			query := skydb.Query{
				Type: "note",
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record2)
			So(records[1], ShouldResemble, record1)
			So(records[2], ShouldResemble, record3)
			So(len(records), ShouldEqual, 3)
		})

		Convey("sorts queried records ascendingly", func() {
			query := skydb.Query{
				Type: "note",
				Sorts: []skydb.Sort{
					skydb.Sort{
						KeyPath: "noteOrder",
						Order:   skydb.Ascending,
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{
				record1,
				record2,
				record3,
			})
		})

		Convey("sorts queried records descendingly", func() {
			query := skydb.Query{
				Type: "note",
				Sorts: []skydb.Sort{
					skydb.Sort{
						KeyPath: "noteOrder",
						Order:   skydb.Descending,
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{
				record3,
				record2,
				record1,
			})
		})

		Convey("query records by note order", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "noteOrder",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: 1,
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record1)
			So(len(records), ShouldEqual, 1)
		})

		Convey("query records by content matching", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Like,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "content",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: "Hello%",
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record1)
			So(len(records), ShouldEqual, 1)
		})

		Convey("query records by case insensitive content matching", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.ILike,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "content",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: "hello%",
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record1)
			So(len(records), ShouldEqual, 1)
		})

		Convey("query records by check array members", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.In,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "content",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: []interface{}{"Bye World", "Good Hello", "Anything"},
						},
					},
				},
				Sorts: []skydb.Sort{
					skydb.Sort{
						KeyPath: "noteOrder",
						Order:   skydb.Descending,
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record3)
			So(records[1], ShouldResemble, record2)
			So(len(records), ShouldEqual, 2)
		})

		Convey("query records by checking empty array", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.In,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "content",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: []interface{}{},
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 0)
		})

		Convey("query records by note order using or predicate", func() {
			keyPathExpr := skydb.Expression{
				Type:  skydb.KeyPath,
				Value: "noteOrder",
			}
			value1 := skydb.Expression{
				Type:  skydb.Literal,
				Value: 2,
			}
			value2 := skydb.Expression{
				Type:  skydb.Literal,
				Value: 3,
			}
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Or,
					Children: []interface{}{
						skydb.Predicate{
							Operator: skydb.Equal,
							Children: []interface{}{keyPathExpr, value1},
						},
						skydb.Predicate{
							Operator: skydb.Equal,
							Children: []interface{}{keyPathExpr, value2},
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record2)
			So(records[1], ShouldResemble, record3)
			So(len(records), ShouldEqual, 2)
		})

		Convey("query records by offset and paging", func() {
			query := skydb.Query{
				Type:   "note",
				Limit:  new(uint64),
				Offset: 1,
				Sorts: []skydb.Sort{
					skydb.Sort{
						KeyPath: "noteOrder",
						Order:   skydb.Descending,
					},
				},
			}
			*query.Limit = 2
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record2)
			So(records[1], ShouldResemble, record1)
			So(len(records), ShouldEqual, 2)
		})

		Convey("query records for nil item", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "emotion",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: nil,
						},
					},
				},
				Sorts: []skydb.Sort{
					skydb.Sort{
						KeyPath: "noteOrder",
						Order:   skydb.Ascending,
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record1)
			So(records[1], ShouldResemble, record2)
			So(len(records), ShouldEqual, 2)
		})

		Convey("query records for not nil item", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.NotEqual,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "emotion",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: nil,
						},
					},
				},
				Sorts: []skydb.Sort{
					skydb.Sort{
						KeyPath: "noteOrder",
						Order:   skydb.Ascending,
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, record3)
			So(len(records), ShouldEqual, 1)
		})
	})

	Convey("Database with reference", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		// fixture
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id1"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(1),
			},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id2"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(2),
				"category":  skydb.NewReference("category", "important"),
			},
		}
		record3 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id3"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(3),
				"category":  skydb.NewReference("category", "funny"),
			},
		}
		category1 := skydb.Record{
			ID:      skydb.NewRecordID("category", "important"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"hidden": false,
			},
		}
		category2 := skydb.Record{
			ID:      skydb.NewRecordID("category", "funny"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"hidden": true,
			},
		}

		db := c.PrivateDB("userid")
		So(db.Extend("category", skydb.RecordSchema{
			"hidden": skydb.FieldType{Type: skydb.TypeBoolean},
		}), ShouldBeNil)
		So(db.Extend("note", skydb.RecordSchema{
			"noteOrder": skydb.FieldType{Type: skydb.TypeNumber},
			"category": skydb.FieldType{
				Type:          skydb.TypeReference,
				ReferenceType: "category",
			},
		}), ShouldBeNil)

		err := db.Save(&category1)
		So(err, ShouldBeNil)
		err = db.Save(&category2)
		So(err, ShouldBeNil)
		err = db.Save(&record2)
		So(err, ShouldBeNil)
		err = db.Save(&record1)
		So(err, ShouldBeNil)
		err = db.Save(&record3)
		So(err, ShouldBeNil)

		Convey("query records by reference", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "category",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: skydb.NewReference("category", "important"),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 1)
			So(records[0], ShouldResemble, record2)
		})

		Convey("query records by comparing field in a referenced record", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "category.hidden",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: true,
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 1)
			So(records[0], ShouldResemble, record3)
		})
	})

	Convey("Database with location", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		record0 := skydb.Record{
			ID:      skydb.NewRecordID("restaurant", "0"),
			OwnerID: "someuserid",
			Data: map[string]interface{}{
				"location": skydb.NewLocation(0, 0),
			},
		}
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("restaurant", "1"),
			OwnerID: "someuserid",
			Data: map[string]interface{}{
				"location": skydb.NewLocation(1, 0),
			},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("restaurant", "2"),
			OwnerID: "someuserid",
			Data: map[string]interface{}{
				"location": skydb.NewLocation(0, 1),
			},
		}

		db := c.PublicDB()
		So(db.Extend("restaurant", skydb.RecordSchema{
			"location": skydb.FieldType{Type: skydb.TypeLocation},
		}), ShouldBeNil)
		So(db.Save(&record0), ShouldBeNil)
		So(db.Save(&record1), ShouldBeNil)
		So(db.Save(&record2), ShouldBeNil)

		Convey("query within distance", func() {
			query := skydb.Query{
				Type: "restaurant",
				Predicate: skydb.Predicate{
					Operator: skydb.LessThanOrEqual,
					Children: []interface{}{
						skydb.Expression{
							Type: skydb.Function,
							Value: skydb.DistanceFunc{
								Field:    "location",
								Location: skydb.NewLocation(1, 1),
							},
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: 157260,
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0, record1, record2})
		})

		Convey("query within distance with func on R.H.S.", func() {
			query := skydb.Query{
				Type: "restaurant",
				Predicate: skydb.Predicate{
					Operator: skydb.GreaterThan,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Literal,
							Value: 157260,
						},
						skydb.Expression{
							Type: skydb.Function,
							Value: skydb.DistanceFunc{
								Field:    "location",
								Location: skydb.NewLocation(1, 1),
							},
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0, record1, record2})
		})

		Convey("query with computed distance", func() {
			query := skydb.Query{
				Type: "restaurant",
				ComputedKeys: map[string]skydb.Expression{
					"distance": skydb.Expression{
						Type: skydb.Function,
						Value: skydb.DistanceFunc{
							Field:    "location",
							Location: skydb.NewLocation(1, 1),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 3)
			So(records[0].Transient["distance"], ShouldAlmostEqual, 157249, 1)
		})

		Convey("query records ordered by distance", func() {
			query := skydb.Query{
				Type: "restaurant",
				Sorts: []skydb.Sort{
					{
						Func: skydb.DistanceFunc{
							Field:    "location",
							Location: skydb.NewLocation(0, 0),
						},
						Order: skydb.Desc,
					},
				},
			}

			records, err := exhaustRows(db.Query(&query))
			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1, record2, record0})
		})
	})

	Convey("Database with multiple fields", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		record0 := skydb.Record{
			ID:      skydb.NewRecordID("restaurant", "0"),
			OwnerID: "someuserid",
			Data: map[string]interface{}{
				"cuisine": "american",
				"title":   "American Restaurant",
			},
		}
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("restaurant", "1"),
			OwnerID: "someuserid",
			Data: map[string]interface{}{
				"cuisine": "chinese",
				"title":   "Chinese Restaurant",
			},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("restaurant", "2"),
			OwnerID: "someuserid",
			Data: map[string]interface{}{
				"cuisine": "italian",
				"title":   "Italian Restaurant",
			},
		}

		recordsInDB := []skydb.Record{record0, record1, record2}

		db := c.PublicDB()
		So(db.Extend("restaurant", skydb.RecordSchema{
			"title":   skydb.FieldType{Type: skydb.TypeString},
			"cuisine": skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)
		So(db.Save(&record0), ShouldBeNil)
		So(db.Save(&record1), ShouldBeNil)
		So(db.Save(&record2), ShouldBeNil)

		Convey("query with desired keys", func() {
			query := skydb.Query{
				Type:        "restaurant",
				DesiredKeys: []string{"cuisine"},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 3)
			for i, record := range records {
				So(record.Data["title"], ShouldBeNil)
				So(record.Data["cuisine"], ShouldEqual, recordsInDB[i].Data["cuisine"])
			}
		})

		Convey("query with empty desired keys", func() {
			query := skydb.Query{
				Type:        "restaurant",
				DesiredKeys: []string{},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 3)
			for _, record := range records {
				So(record.Data["title"], ShouldBeNil)
				So(record.Data["cuisine"], ShouldBeNil)
			}
		})

		Convey("query with nil desired keys", func() {
			query := skydb.Query{
				Type:        "restaurant",
				DesiredKeys: nil,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 3)
			for i, record := range records {
				So(record.Data["title"], ShouldEqual, recordsInDB[i].Data["title"])
				So(record.Data["cuisine"], ShouldEqual, recordsInDB[i].Data["cuisine"])
			}
		})

		Convey("query with non-recognized desired keys", func() {
			query := skydb.Query{
				Type:        "restaurant",
				DesiredKeys: []string{"pricing"},
			}
			_, err := exhaustRows(db.Query(&query))

			So(err, ShouldNotBeNil)
		})
	})

	Convey("Database with JSON", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		// fixture
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id1"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"primaryTag": "red",
				"tags":       []interface{}{"red", "green"},
			},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id2"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"primaryTag": "yellow",
				"tags":       []interface{}{"red", "green"},
			},
		}
		record3 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id3"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"primaryTag": "green",
				"tags":       []interface{}{"red", "yellow"},
			},
		}

		db := c.PrivateDB("userid")
		So(db.Extend("note", skydb.RecordSchema{
			"primaryTag": skydb.FieldType{Type: skydb.TypeString},
			"tags":       skydb.FieldType{Type: skydb.TypeJSON},
		}), ShouldBeNil)

		err := db.Save(&record2)
		So(err, ShouldBeNil)
		err = db.Save(&record1)
		So(err, ShouldBeNil)
		err = db.Save(&record3)
		So(err, ShouldBeNil)

		Convey("query records by literal string in JSON", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.In,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Literal,
							Value: "yellow",
						},
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "tags",
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record3})
		})
	})

	Convey("Database with ACL", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		// fixture
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id1"),
			OwnerID: "alice",
			ACL:     skydb.RecordACL{},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id2"),
			OwnerID: "alice",
			ACL: skydb.RecordACL{
				skydb.NewRecordACLEntryPublic(skydb.ReadLevel),
			},
		}
		record3 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id3"),
			OwnerID: "alice",
			ACL: skydb.RecordACL{
				skydb.NewRecordACLEntryDirect("bob", skydb.ReadLevel),
			},
		}
		record4 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id4"),
			OwnerID: "alice",
			ACL: skydb.RecordACL{
				skydb.NewRecordACLEntryRole("marketing", skydb.ReadLevel),
			},
		}
		record5 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id5"),
			OwnerID: "alice",
			ACL: skydb.RecordACL{
				skydb.NewRecordACLEntryDirect("bob", skydb.ReadLevel),
				skydb.NewRecordACLEntryRole("marketing", skydb.ReadLevel),
			},
		}

		db := c.PublicDB()
		So(db.Extend("note", skydb.RecordSchema{}), ShouldBeNil)

		err := db.Save(&record1)
		So(err, ShouldBeNil)
		err = db.Save(&record2)
		So(err, ShouldBeNil)
		err = db.Save(&record3)
		So(err, ShouldBeNil)
		err = db.Save(&record4)
		So(err, ShouldBeNil)
		err = db.Save(&record5)
		So(err, ShouldBeNil)

		sortsByID := []skydb.Sort{
			skydb.Sort{
				KeyPath: "_id",
				Order:   skydb.Ascending,
			},
		}

		Convey("can be queried by owner", func() {
			query := skydb.Query{
				Type:       "note",
				ViewAsUser: &skydb.UserInfo{ID: "alice"},
				Sorts:      sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1, record2, record3, record4, record5})
		})

		Convey("can be queried by public", func() {
			query := skydb.Query{
				Type:       "note",
				ViewAsUser: nil,
				Sorts:      sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record2})
		})

		Convey("can be queried by explicit user", func() {
			query := skydb.Query{
				Type:       "note",
				ViewAsUser: &skydb.UserInfo{ID: "bob"},
				Sorts:      sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record2, record3, record5})
		})

		Convey("can be queried by explicit role", func() {
			query := skydb.Query{
				Type: "note",
				ViewAsUser: &skydb.UserInfo{
					ID:    "carol",
					Roles: []string{"marketing"},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record2, record4, record5})
		})

		Convey("can be queried by explicit user and role", func() {
			query := skydb.Query{
				Type: "note",
				ViewAsUser: &skydb.UserInfo{
					ID:    "bob",
					Roles: []string{"marketing"},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record2, record3, record4, record5})
		})

		Convey("can be queried with bypass access control", func() {
			query := skydb.Query{
				Type: "note",
				ViewAsUser: &skydb.UserInfo{
					ID: "dave",
				},
				Sorts:               sortsByID,
				BypassAccessControl: true,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1, record2, record3, record4, record5})
		})
	})

	Convey("Empty Conn", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		Convey("gets no users", func() {
			userinfo := skydb.UserInfo{}
			err := c.GetUser("notexistuserid", &userinfo)
			So(err, ShouldEqual, skydb.ErrUserNotFound)
		})

		Convey("gets no users with principal", func() {
			userinfo := skydb.UserInfo{}
			err := c.GetUserByPrincipalID("com.example:johndoe", &userinfo)
			So(err, ShouldEqual, skydb.ErrUserNotFound)
		})

		Convey("query no users", func() {
			emails := []string{"user@example.com"}
			result, err := c.QueryUser(emails, []string{})
			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 0)
		})

		Convey("updates no users", func() {
			userinfo := skydb.UserInfo{
				ID: "notexistuserid",
			}
			err := c.UpdateUser(&userinfo)
			So(err, ShouldEqual, skydb.ErrUserNotFound)
		})

		Convey("deletes no users", func() {
			err := c.DeleteUser("notexistuserid")
			So(err, ShouldEqual, skydb.ErrUserNotFound)
		})

		Convey("gets no devices", func() {
			device := skydb.Device{}
			err := c.GetDevice("notexistdeviceid", &device)
			So(err, ShouldEqual, skydb.ErrDeviceNotFound)
		})

		Convey("deletes no devices", func() {
			err := c.DeleteDevice("notexistdeviceid")
			So(err, ShouldEqual, skydb.ErrDeviceNotFound)
		})

		Convey("Empty Database", func() {
			db := c.PublicDB()

			Convey("gets nothing", func() {
				record := skydb.Record{}

				err := db.Get(skydb.NewRecordID("type", "notexistid"), &record)

				So(err, ShouldEqual, skydb.ErrRecordNotFound)
			})

			Convey("deletes nothing", func() {
				err := db.Delete(skydb.NewRecordID("type", "notexistid"))
				So(err, ShouldEqual, skydb.ErrRecordNotFound)
			})

			Convey("queries nothing", func() {
				query := skydb.Query{
					Type: "notexisttype",
				}

				records, err := exhaustRows(db.Query(&query))

				So(err, ShouldBeNil)
				So(records, ShouldBeEmpty)
			})
		})
	})
}

func TestQueryCount(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		// fixture
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id1"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(1),
				"content":   "Hello World",
			},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id2"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(2),
				"content":   "Bye World",
			},
		}
		record3 := skydb.Record{
			ID:      skydb.NewRecordID("note", "id3"),
			OwnerID: "user_id",
			Data: map[string]interface{}{
				"noteOrder": float64(3),
				"content":   "Good Hello",
			},
		}

		db := c.PrivateDB("userid")
		So(db.Extend("note", skydb.RecordSchema{
			"noteOrder": skydb.FieldType{Type: skydb.TypeNumber},
			"content":   skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)

		err := db.Save(&record2)
		So(err, ShouldBeNil)
		err = db.Save(&record1)
		So(err, ShouldBeNil)
		err = db.Save(&record3)
		So(err, ShouldBeNil)

		Convey("count records", func() {
			query := skydb.Query{
				Type: "note",
			}
			count, err := db.QueryCount(&query)

			So(err, ShouldBeNil)
			So(count, ShouldEqual, 3)
		})

		Convey("count records by content matching", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Like,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "content",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: "Hello%",
						},
					},
				},
			}
			count, err := db.QueryCount(&query)

			So(err, ShouldBeNil)
			So(count, ShouldEqual, 1)
		})

		Convey("count records by content with none matching", func() {
			query := skydb.Query{
				Type: "note",
				Predicate: skydb.Predicate{
					Operator: skydb.Like,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "content",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: "Not Exist",
						},
					},
				},
			}
			count, err := db.QueryCount(&query)

			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)
		})
	})
}

func TestAggregateQuery(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		// fixture
		db := c.PrivateDB("userid")
		So(db.Extend("note", skydb.RecordSchema{
			"category": skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)

		categories := []string{"funny", "funny", "serious"}
		dbRecords := []skydb.Record{}

		for i, category := range categories {
			record := skydb.Record{
				ID:      skydb.NewRecordID("note", fmt.Sprintf("id%d", i)),
				OwnerID: "user_id",
				Data: map[string]interface{}{
					"category": category,
				},
			}
			err := db.Save(&record)
			dbRecords = append(dbRecords, record)
			So(err, ShouldBeNil)
		}

		equalCategoryPredicate := func(category string) skydb.Predicate {
			return skydb.Predicate{
				Operator: skydb.Equal,
				Children: []interface{}{
					skydb.Expression{
						Type:  skydb.KeyPath,
						Value: "category",
					},
					skydb.Expression{
						Type:  skydb.Literal,
						Value: category,
					},
				},
			}
		}

		Convey("queries records", func() {
			query := skydb.Query{
				Type:      "note",
				Predicate: equalCategoryPredicate("funny"),
				GetCount:  true,
			}
			rows, err := db.Query(&query)
			records, err := exhaustRows(rows, err)

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 2)
			So(records[0], ShouldResemble, dbRecords[0])
			So(records[1], ShouldResemble, dbRecords[1])

			recordCount := rows.OverallRecordCount()
			So(recordCount, ShouldNotBeNil)
			So(*recordCount, ShouldEqual, 2)
		})

		Convey("queries no records", func() {
			query := skydb.Query{
				Type:      "note",
				Predicate: equalCategoryPredicate("interesting"),
				GetCount:  true,
			}
			rows, err := db.Query(&query)
			records, err := exhaustRows(rows, err)

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 0)

			recordCount := rows.OverallRecordCount()
			So(recordCount, ShouldBeNil)
		})

		Convey("queries records with limit", func() {
			query := skydb.Query{
				Type:      "note",
				Predicate: equalCategoryPredicate("funny"),
				GetCount:  true,
				Limit:     new(uint64),
			}
			*query.Limit = 1
			rows, err := db.Query(&query)
			records, err := exhaustRows(rows, err)

			So(err, ShouldBeNil)
			So(records[0], ShouldResemble, dbRecords[0])
			So(len(records), ShouldEqual, 1)

			recordCount := rows.OverallRecordCount()
			So(recordCount, ShouldNotBeNil)
			So(*recordCount, ShouldEqual, 2)
		})
	})
}

func TestMetaDataQuery(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		record0 := skydb.Record{
			ID:        skydb.NewRecordID("record", "0"),
			OwnerID:   "ownerID0",
			CreatedAt: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			CreatorID: "creatorID0",
			UpdatedAt: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			UpdaterID: "updaterID0",
			Data:      skydb.Data{},
		}
		record1 := skydb.Record{
			ID:        skydb.NewRecordID("record", "1"),
			OwnerID:   "ownerID1",
			CreatedAt: time.Date(2006, 1, 2, 15, 4, 6, 0, time.UTC),
			CreatorID: "creatorID1",
			UpdatedAt: time.Date(2006, 1, 2, 15, 4, 6, 0, time.UTC),
			UpdaterID: "updaterID1",
			Data:      skydb.Data{},
		}

		db := c.PublicDB()
		So(db.Extend("record", nil), ShouldBeNil)
		So(db.Save(&record0), ShouldBeNil)
		So(db.Save(&record1), ShouldBeNil)

		Convey("queries by record id", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "_id",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: skydb.NewReference("record", "0"),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0})
		})

		Convey("queries by owner id", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "_owner_id",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: skydb.NewReference("_user", "ownerID1"),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1})
		})

		Convey("queries by created at", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.LessThan,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "_created_at",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: time.Date(2006, 1, 2, 15, 4, 6, 0, time.UTC),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0})
		})

		Convey("queries by created by", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "_created_by",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: skydb.NewReference("_user", "creatorID0"),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0})
		})

		Convey("queries by updated at", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.GreaterThan,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "_updated_at",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1})
		})

		Convey("queries by updated by", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Equal,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "_updated_by",
						},
						skydb.Expression{
							Type:  skydb.Literal,
							Value: skydb.NewReference("_user", "updaterID1"),
						},
					},
				},
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1})
		})
	})
}

func TestUserRelationQuery(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		addUser(t, c, "user1")
		addUser(t, c, "user2") // followed by user1
		addUser(t, c, "user3") // mutual follower of user1
		addUser(t, c, "user4") // friend of user1
		addUser(t, c, "user5") // friend of user4 and followed by user4
		c.AddRelation("user1", "_follow", "user2")
		c.AddRelation("user1", "_follow", "user3")
		c.AddRelation("user3", "_follow", "user1")
		c.AddRelation("user1", "_friend", "user4")
		c.AddRelation("user4", "_friend", "user1")
		c.AddRelation("user4", "_friend", "user5")
		c.AddRelation("user5", "_friend", "user4")
		c.AddRelation("user4", "_follow", "user5")

		record0 := skydb.Record{
			ID:      skydb.NewRecordID("record", "0"),
			OwnerID: "user1",
			Data:    skydb.Data{},
		}
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("record", "1"),
			OwnerID: "user2",
			Data:    skydb.Data{},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("record", "2"),
			OwnerID: "user3",
			Data:    skydb.Data{},
		}
		record3 := skydb.Record{
			ID:      skydb.NewRecordID("record", "3"),
			OwnerID: "user4",
			Data:    skydb.Data{},
		}
		record4 := skydb.Record{
			ID:      skydb.NewRecordID("record", "4"),
			OwnerID: "user5",
			Data:    skydb.Data{},
		}

		db := c.PublicDB()
		So(db.Extend("record", nil), ShouldBeNil)
		So(db.Save(&record0), ShouldBeNil)
		So(db.Save(&record1), ShouldBeNil)
		So(db.Save(&record2), ShouldBeNil)
		So(db.Save(&record3), ShouldBeNil)
		So(db.Save(&record4), ShouldBeNil)

		sortsByID := []skydb.Sort{
			skydb.Sort{
				KeyPath: "_id",
				Order:   skydb.Ascending,
			},
		}

		Convey("query follow outward", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Function,
							Value: skydb.UserRelationFunc{"_owner", "_follow", "outward", "user1"},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record1, record2})
		})

		Convey("query follow inward", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Function,
							Value: skydb.UserRelationFunc{"_owner", "_follow", "inward", "user2"},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0})
		})

		Convey("query follow outward OR inward", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Or,
					Children: []interface{}{
						skydb.Predicate{
							Operator: skydb.Functional,
							Children: []interface{}{
								skydb.Expression{
									Type:  skydb.Function,
									Value: skydb.UserRelationFunc{"_owner", "_follow", "outward", "user1"},
								},
							},
						},
						skydb.Predicate{
							Operator: skydb.Functional,
							Children: []interface{}{
								skydb.Expression{
									Type:  skydb.Function,
									Value: skydb.UserRelationFunc{"_owner", "_follow", "inward", "user2"},
								},
							},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0, record1, record2})
		})

		Convey("query follow mutual for user1", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Function,
							Value: skydb.UserRelationFunc{"_owner", "_follow", "mutual", "user1"},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record2})
		})

		Convey("query follow mutual for user2", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Function,
							Value: skydb.UserRelationFunc{"_owner", "_follow", "mutual", "user2"},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 0)
		})

		Convey("query friend mutual", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Function,
							Value: skydb.UserRelationFunc{"_owner", "_friend", "mutual", "user1"},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record3})
		})

		Convey("distinct record satisfying both relations", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.Or,
					Children: []interface{}{
						skydb.Predicate{
							Operator: skydb.Functional,
							Children: []interface{}{
								skydb.Expression{
									Type:  skydb.Function,
									Value: skydb.UserRelationFunc{"_owner", "_follow", "outward", "user4"},
								},
							},
						},
						skydb.Predicate{
							Operator: skydb.Functional,
							Children: []interface{}{
								skydb.Expression{
									Type:  skydb.Function,
									Value: skydb.UserRelationFunc{"_owner", "_friend", "mutual", "user4"},
								},
							},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))
			So(err, ShouldBeNil)
			So(records, ShouldResemble, []skydb.Record{record0, record4})

			count, err := db.QueryCount(&query)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 2)
		})
	})
}

func TestUserDiscoverQuery(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		addUserWithInfo(t, c, "user0", "john.doe@example.com")
		addUserWithInfo(t, c, "user1", "jane.doe@example.com")
		addUserWithUsername(t, c, "user2", "john.doe")

		record0 := skydb.Record{
			ID:      skydb.NewRecordID("user", "user0"),
			OwnerID: "user0",
			Data:    skydb.Data{},
		}
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("user", "user1"),
			OwnerID: "user1",
			Data:    skydb.Data{},
		}
		record2 := skydb.Record{
			ID:      skydb.NewRecordID("user", "user2"),
			OwnerID: "user2",
			Data:    skydb.Data{},
		}

		db := c.PublicDB()
		So(db.Extend("user", nil), ShouldBeNil)
		So(db.Save(&record0), ShouldBeNil)
		So(db.Save(&record1), ShouldBeNil)
		So(db.Save(&record2), ShouldBeNil)

		sortsByID := []skydb.Sort{
			skydb.Sort{
				KeyPath: "_id",
				Order:   skydb.Ascending,
			},
		}

		Convey("search single user", func() {
			query := skydb.Query{
				Type: "user",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.Function,
							Value: skydb.UserDiscoverFunc{Emails: []string{"john.doe@example.com"}},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 1)
			So(records[0].ID, ShouldResemble, record0.ID)
			So(records[0].Transient, ShouldResemble, skydb.Data{"_email": "john.doe@example.com"})
		})

		Convey("search multiple user", func() {
			query := skydb.Query{
				Type: "user",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type: skydb.Function,
							Value: skydb.UserDiscoverFunc{
								Emails: []string{"john.doe@example.com", "jane.doe@example.com"},
							},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 2)

			savedRecords := []skydb.Record{record0, record1}
			savedEmails := []string{"john.doe@example.com", "jane.doe@example.com"}
			for i, record := range records {
				So(record.ID, ShouldResemble, savedRecords[i].ID)
				So(record.Transient, ShouldResemble, skydb.Data{"_email": savedEmails[i]})
			}
		})

		Convey("search by username and email", func() {
			query := skydb.Query{
				Type: "user",
				Predicate: skydb.Predicate{
					Operator: skydb.Functional,
					Children: []interface{}{
						skydb.Expression{
							Type: skydb.Function,
							Value: skydb.UserDiscoverFunc{
								Emails:    []string{"john.doe@example.com"},
								Usernames: []string{"john.doe"},
							},
						},
					},
				},
				Sorts: sortsByID,
			}
			records, err := exhaustRows(db.Query(&query))

			So(err, ShouldBeNil)
			So(len(records), ShouldEqual, 2)

			savedRecords := []skydb.Record{record0, record2}

			for i, record := range records {
				So(record.ID, ShouldResemble, savedRecords[i].ID)
			}
		})
	})
}

func TestUnsupportedQuery(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		record0 := skydb.Record{
			ID:      skydb.NewRecordID("record", "0"),
			OwnerID: "ownerID0",
			Data:    skydb.Data{},
		}
		record1 := skydb.Record{
			ID:      skydb.NewRecordID("record", "1"),
			OwnerID: "ownerID1",
			Data:    skydb.Data{},
		}

		db := c.PublicDB()
		So(db.Extend("record", skydb.RecordSchema{
			"categories":       skydb.FieldType{Type: skydb.TypeString},
			"favoriteCategory": skydb.FieldType{Type: skydb.TypeString},
		}), ShouldBeNil)
		So(db.Save(&record0), ShouldBeNil)
		So(db.Save(&record1), ShouldBeNil)

		Convey("both side of IN is keypath", func() {
			query := skydb.Query{
				Type: "record",
				Predicate: skydb.Predicate{
					Operator: skydb.In,
					Children: []interface{}{
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "categories",
						},
						skydb.Expression{
							Type:  skydb.KeyPath,
							Value: "favoriteCategory",
						},
					},
				},
			}
			So(func() { db.Query(&query) }, ShouldPanicWith, "malformed query")
		})
	})
}

func TestRecordACL(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PublicDB()
		So(db.Extend("note", nil), ShouldBeNil)

		record := skydb.Record{
			ID:      skydb.NewRecordID("note", "1"),
			OwnerID: "someuserid",
			ACL:     nil,
		}

		Convey("saves public access correctly", func() {
			err := db.Save(&record)

			So(err, ShouldBeNil)

			var b []byte
			err = c.QueryRowx(`SELECT _access FROM app_com_oursky_skygear.note WHERE _id = '1'`).
				Scan(&b)
			So(err, ShouldBeNil)
			So(b, ShouldResemble, []byte(nil))
		})
	})
}

func TestRecordJSON(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PublicDB()
		So(db.Extend("note", skydb.RecordSchema{
			"jsonfield": skydb.FieldType{Type: skydb.TypeJSON},
		}), ShouldBeNil)

		Convey("fetch record with json field", func() {
			So(db.Extend("record", skydb.RecordSchema{
				"array":      skydb.FieldType{Type: skydb.TypeJSON},
				"dictionary": skydb.FieldType{Type: skydb.TypeJSON},
			}), ShouldBeNil)

			insertRow(t, c.Db(), `INSERT INTO app_com_oursky_skygear."record" `+
				`(_database_id, _id, _owner_id, _created_at, _created_by, _updated_at, _updated_by, "array", "dictionary") `+
				`VALUES ('', 'id', '', '0001-01-01 00:00:00', '', '0001-01-01 00:00:00', '', '[1, "string", true]', '{"number": 0, "string": "value", "bool": false}')`)

			var record skydb.Record
			err := db.Get(skydb.NewRecordID("record", "id"), &record)
			So(err, ShouldBeNil)

			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("record", "id"),
				Data: map[string]interface{}{
					"array": []interface{}{float64(1), "string", true},
					"dictionary": map[string]interface{}{
						"number": float64(0),
						"string": "value",
						"bool":   false,
					},
				},
			})
		})

		Convey("saves record field with array", func() {
			record := skydb.Record{
				ID:      skydb.NewRecordID("note", "1"),
				OwnerID: "user_id",
				Data: map[string]interface{}{
					"jsonfield": []interface{}{0.0, "string", true},
				},
			}

			So(db.Save(&record), ShouldBeNil)

			var jsonBytes []byte
			err := c.QueryRowx(`SELECT jsonfield FROM app_com_oursky_skygear.note WHERE _id = '1' and _database_id = ''`).
				Scan(&jsonBytes)
			So(err, ShouldBeNil)
			So(jsonBytes, ShouldEqualJSON, `[0, "string", true]`)
		})

		Convey("saves record field with dictionary", func() {
			record := skydb.Record{
				ID:      skydb.NewRecordID("note", "1"),
				OwnerID: "user_id",
				Data: map[string]interface{}{
					"jsonfield": map[string]interface{}{
						"number": float64(1),
						"string": "",
						"bool":   false,
					},
				},
			}

			So(db.Save(&record), ShouldBeNil)

			var jsonBytes []byte
			err := c.QueryRowx(`SELECT jsonfield FROM app_com_oursky_skygear.note WHERE _id = '1' and _database_id = ''`).
				Scan(&jsonBytes)
			So(err, ShouldBeNil)
			So(jsonBytes, ShouldEqualJSON, `{"number": 1, "string": "", "bool": false}`)
		})
	})
}

func TestRecordAssetField(t *testing.T) {
	Convey("Record Asset", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		So(c.SaveAsset(&skydb.Asset{
			Name:        "picture.png",
			ContentType: "image/png",
			Size:        1,
		}), ShouldBeNil)

		db := c.PublicDB()
		So(db.Extend("note", skydb.RecordSchema{
			"image": skydb.FieldType{Type: skydb.TypeAsset},
		}), ShouldBeNil)

		Convey("can be associated", func() {
			err := db.Save(&skydb.Record{
				ID: skydb.NewRecordID("note", "id"),
				Data: map[string]interface{}{
					"image": &skydb.Asset{Name: "picture.png"},
				},
				OwnerID: "user_id",
			})
			So(err, ShouldBeNil)
		})

		Convey("errors when associated with non-existing asset", func() {
			err := db.Save(&skydb.Record{
				ID: skydb.NewRecordID("note", "id"),
				Data: map[string]interface{}{
					"image": &skydb.Asset{Name: "notexist.png"},
				},
				OwnerID: "user_id",
			})
			So(err, ShouldNotBeNil)
		})

		Convey("REGRESSION #229: can be fetched", func() {
			So(db.Save(&skydb.Record{
				ID: skydb.NewRecordID("note", "id"),
				Data: map[string]interface{}{
					"image": &skydb.Asset{Name: "picture.png"},
				},
				OwnerID: "user_id",
			}), ShouldBeNil)

			var record skydb.Record
			err := db.Get(skydb.NewRecordID("note", "id"), &record)
			So(err, ShouldBeNil)
			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("note", "id"),
				Data: map[string]interface{}{
					"image": &skydb.Asset{Name: "picture.png"},
				},
				OwnerID: "user_id",
			})
		})
	})
}

func TestRecordLocationField(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PublicDB()
		So(db.Extend("photo", skydb.RecordSchema{
			"location": skydb.FieldType{Type: skydb.TypeLocation},
		}), ShouldBeNil)

		Convey("saves & load location field", func() {
			err := db.Save(&skydb.Record{
				ID: skydb.NewRecordID("photo", "1"),
				Data: map[string]interface{}{
					"location": skydb.NewLocation(1, 2),
				},
				OwnerID: "userid",
			})

			So(err, ShouldBeNil)

			record := skydb.Record{}
			err = db.Get(skydb.NewRecordID("photo", "1"), &record)
			So(err, ShouldBeNil)
			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("photo", "1"),
				Data: map[string]interface{}{
					"location": skydb.NewLocation(1, 2),
				},
				OwnerID: "userid",
			})
		})
	})
}

func TestRecordSequenceField(t *testing.T) {
	Convey("Database", t, func() {
		c := getTestConn(t)
		defer cleanupConn(t, c)

		db := c.PublicDB()
		So(db.Extend("note", skydb.RecordSchema{
			"seq": skydb.FieldType{Type: skydb.TypeSequence},
		}), ShouldBeNil)

		Convey("saves & load sequence field", func() {
			record := skydb.Record{
				ID:      skydb.NewRecordID("note", "1"),
				OwnerID: "userid",
			}

			err := db.Save(&record)
			So(err, ShouldBeNil)
			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("note", "1"),
				Data: map[string]interface{}{
					"seq": int64(1),
				},
				OwnerID: "userid",
			})

			record = skydb.Record{
				ID:      skydb.NewRecordID("note", "2"),
				OwnerID: "userid",
			}

			err = db.Save(&record)
			So(err, ShouldBeNil)
			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("note", "2"),
				Data: map[string]interface{}{
					"seq": int64(2),
				},
				OwnerID: "userid",
			})
		})

		Convey("updates sequence field manually", func() {
			record := skydb.Record{
				ID:      skydb.NewRecordID("note", "1"),
				OwnerID: "userid",
			}

			So(db.Save(&record), ShouldBeNil)
			So(record.Data["seq"], ShouldEqual, 1)

			record.Data["seq"] = 10
			So(db.Save(&record), ShouldBeNil)

			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("note", "1"),
				Data: map[string]interface{}{
					"seq": int64(10),
				},
				OwnerID: "userid",
			})

			// next record should's seq value should be 11
			record = skydb.Record{
				ID:      skydb.NewRecordID("note", "2"),
				OwnerID: "userid",
			}
			So(db.Save(&record), ShouldBeNil)
			So(record, ShouldResemble, skydb.Record{
				ID: skydb.NewRecordID("note", "2"),
				Data: map[string]interface{}{
					"seq": int64(11),
				},
				OwnerID: "userid",
			})
		})
	})
}
