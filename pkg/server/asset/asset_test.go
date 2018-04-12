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

package asset

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPresignFunctions(t *testing.T) {

	Convey("Presign Interval", t, func() {
		Convey("should calculate start time", func() {
			now := time.Now()
			diff := time.Minute * time.Duration(15)
			startTime := getPresignIntervalStartTime(now, diff)
			So(startTime.Sub(now), ShouldEqual, diff)
		})
	})

	Convey("Presign Expire", t, func() {
		Convey("should calculate expire time", func() {
			now := time.Now()
			diff := time.Minute * time.Duration(15)
			expire := time.Minute * time.Duration(5)
			expireTime := getPresignExpireTime(now, diff, expire)
			So(expireTime.Sub(now), ShouldEqual, diff)
		})
	})
}
