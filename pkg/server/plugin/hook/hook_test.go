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

package hook

import (
	"testing"

	"github.com/skygeario/skygear-server/pkg/server/plugin/hook/hooktest"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

type ContextKey string

var HelloContextKey ContextKey = "hello"

func TestHookRegistry(t *testing.T) {
	Convey("Registry", t, func() {

		beforeSave := hooktest.StackingHook{}
		afterSave := hooktest.StackingHook{}
		beforeDelete := hooktest.StackingHook{}
		afterDelete := hooktest.StackingHook{}
		ctx := context.WithValue(context.Background(), HelloContextKey, "world")

		registry := NewRegistry()

		Convey("executes hooks once", func() {
			registry.Register(BeforeSave, "record", beforeSave.Func)
			registry.Register(AfterSave, "record", afterSave.Func)
			registry.Register(BeforeDelete, "record", beforeDelete.Func)
			registry.Register(AfterDelete, "record", afterDelete.Func)

			record := &skydb.Record{
				ID: skydb.NewRecordID("record", "id"),
			}

			originalRecord := &skydb.Record{
				ID: record.ID,
				Data: skydb.Data{
					"value": "old",
				},
			}

			Convey("for beforeSave", func() {
				registry.ExecuteHooks(ctx, BeforeSave, record, originalRecord)
				So(beforeSave.Records, ShouldResemble, []*skydb.Record{record})
				So(beforeSave.OriginalRecords, ShouldResemble, []*skydb.Record{originalRecord})
				So(beforeSave.Context[0].Value(HelloContextKey), ShouldEqual, "world")
				So(afterSave.Records, ShouldBeEmpty)
				So(afterSave.OriginalRecords, ShouldBeEmpty)
				So(beforeDelete.Records, ShouldBeEmpty)
				So(afterDelete.Records, ShouldBeEmpty)
			})

			Convey("for afterSave", func() {
				registry.ExecuteHooks(ctx, AfterSave, record, originalRecord)
				So(beforeSave.Records, ShouldBeEmpty)
				So(beforeSave.OriginalRecords, ShouldBeEmpty)
				So(afterSave.Records, ShouldResemble, []*skydb.Record{record})
				So(afterSave.OriginalRecords, ShouldResemble, []*skydb.Record{originalRecord})
				So(afterSave.Context[0].Value(HelloContextKey), ShouldEqual, "world")
				So(beforeDelete.Records, ShouldBeEmpty)
				So(afterDelete.Records, ShouldBeEmpty)
			})

			Convey("for beforeDelete", func() {
				registry.ExecuteHooks(ctx, BeforeDelete, record, originalRecord)
				So(beforeSave.Records, ShouldBeEmpty)
				So(beforeSave.OriginalRecords, ShouldBeEmpty)
				So(afterSave.Records, ShouldBeEmpty)
				So(afterSave.OriginalRecords, ShouldBeEmpty)
				So(beforeDelete.Records, ShouldResemble, []*skydb.Record{record})
				So(beforeDelete.Context[0].Value(HelloContextKey), ShouldEqual, "world")
				So(afterDelete.Records, ShouldBeEmpty)
			})

			Convey("for afterDelete", func() {
				registry.ExecuteHooks(ctx, AfterDelete, record, originalRecord)
				So(beforeSave.Records, ShouldBeEmpty)
				So(beforeSave.OriginalRecords, ShouldBeEmpty)
				So(afterSave.Records, ShouldBeEmpty)
				So(afterSave.OriginalRecords, ShouldBeEmpty)
				So(beforeDelete.Records, ShouldBeEmpty)
				So(afterDelete.Records, ShouldResemble, []*skydb.Record{record})
				So(afterDelete.Context[0].Value(HelloContextKey), ShouldEqual, "world")
			})
		})

		Convey("executes multiple hooks", func() {
			hook1 := hooktest.StackingHook{}
			hook2 := hooktest.StackingHook{}
			registry.Register(AfterSave, "note", hook1.Func)
			registry.Register(AfterSave, "note", hook2.Func)

			record := &skydb.Record{
				ID: skydb.NewRecordID("note", "id"),
			}
			originalRecord := &skydb.Record{
				ID: record.ID,
				Data: skydb.Data{
					"value": "old",
				},
			}
			registry.ExecuteHooks(ctx, AfterSave, record, originalRecord)

			So(hook1.Records, ShouldResemble, []*skydb.Record{record})
			So(hook2.Records, ShouldResemble, []*skydb.Record{record})
			So(hook1.OriginalRecords, ShouldResemble, []*skydb.Record{originalRecord})
			So(hook2.OriginalRecords, ShouldResemble, []*skydb.Record{originalRecord})
			So(hook1.Context[0].Value(HelloContextKey), ShouldEqual, "world")
			So(hook2.Context[0].Value(HelloContextKey), ShouldEqual, "world")
		})

		Convey("executes no hooks", func() {
			record := &skydb.Record{
				ID: skydb.NewRecordID("record", "id"),
			}
			So(func() {
				registry.ExecuteHooks(ctx, BeforeDelete, record, nil)
			}, ShouldNotPanic)
		})

		Convey("panics executing nil record", func() {
			So(func() {
				registry.ExecuteHooks(ctx, AfterDelete, nil, nil)
			}, ShouldPanic)
		})
	})
}
