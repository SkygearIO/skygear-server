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

package captcha

import (
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type tokenBucketStore struct {
	data map[string]TokenBucketInfo
}

func (s *tokenBucketStore) Get(id string) (TokenBucketInfo, bool, error) {
	info, ok := s.data[id]
	return info, ok, nil
}

func (s *tokenBucketStore) Set(id string, info TokenBucketInfo) error {
	if s.data == nil {
		s.data = map[string]TokenBucketInfo{}
	}
	s.data[id] = info
	return nil
}

func TestTokenBucket(t *testing.T) {
	// 6 requests per 1 minute.
	// The refill rate is 0.1 token per second.
	rate := Rate{
		Limit:    6,
		Duration: 1 * time.Minute,
	}
	store := tokenBucketStore{}
	var rateLimiter RateLimiter
	tokenBucket := TokenBucket{
		Store: &store,
		Rate:  rate,
	}
	// Ensure TokenBucket implements RateLimiter
	rateLimiter = &tokenBucket
	timeline := []struct {
		expected bool
		t        time.Time
	}{
		// t = 0
		{true, time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		// t = 1
		{true, time.Date(2018, 1, 1, 0, 0, 1, 0, time.UTC)},
		// t = 2
		{true, time.Date(2018, 1, 1, 0, 0, 2, 0, time.UTC)},
		// t = 3
		{true, time.Date(2018, 1, 1, 0, 0, 3, 0, time.UTC)},
		// t = 4
		{true, time.Date(2018, 1, 1, 0, 0, 4, 0, time.UTC)},
		// t = 5
		{true, time.Date(2018, 1, 1, 0, 0, 5, 0, time.UTC)},
		// t = 6
		{false, time.Date(2018, 1, 1, 0, 0, 6, 0, time.UTC)},
		// t = 7
		{false, time.Date(2018, 1, 1, 0, 0, 7, 0, time.UTC)},
		// t = 8
		{false, time.Date(2018, 1, 1, 0, 0, 8, 0, time.UTC)},
		// t = 9
		{false, time.Date(2018, 1, 1, 0, 0, 9, 0, time.UTC)},
		// t = 10.0001; 10 seconds elapsed, 1 token was refilled
		{true, time.Date(2018, 1, 1, 0, 0, 10, 1000, time.UTC)},
		// t = 71; 1 minute elapsed, 6 tokens was refilled
		{true, time.Date(2018, 1, 1, 0, 0, 71, 0, time.UTC)},
		// t = 72
		{true, time.Date(2018, 1, 1, 0, 0, 72, 0, time.UTC)},
		// t = 73
		{true, time.Date(2018, 1, 1, 0, 0, 73, 0, time.UTC)},
		// t = 74
		{true, time.Date(2018, 1, 1, 0, 0, 74, 0, time.UTC)},
		// t = 75
		{true, time.Date(2018, 1, 1, 0, 0, 75, 0, time.UTC)},
		// t = 76
		{true, time.Date(2018, 1, 1, 0, 0, 76, 0, time.UTC)},
		// t = 76.0001
		{false, time.Date(2018, 1, 1, 0, 0, 76, 1000, time.UTC)},
		// t = 3677; 1 hour elapsed, 6 tokens was refilled
		{true, time.Date(2018, 1, 1, 0, 0, 3677, 0, time.UTC)},
		// t = 3678
		{true, time.Date(2018, 1, 1, 0, 0, 3678, 0, time.UTC)},
		// t = 3679
		{true, time.Date(2018, 1, 1, 0, 0, 3679, 0, time.UTC)},
		// t = 3680
		{true, time.Date(2018, 1, 1, 0, 0, 3680, 0, time.UTC)},
		// t = 3681
		{true, time.Date(2018, 1, 1, 0, 0, 3681, 0, time.UTC)},
		// t = 3682
		{true, time.Date(2018, 1, 1, 0, 0, 3682, 0, time.UTC)},
		// t = 3682.0001
		{false, time.Date(2018, 1, 1, 0, 0, 3682, 1000, time.UTC)},
	}
	for _, item := range timeline {
		actual, err := rateLimiter.IncrementAt("id", item.t)
		if err != nil {
			t.Error("unexpected error", err)
		}
		if actual != item.expected {
			t.Error("unexpected", actual, item.expected, item.t)
		}
	}
}

func clearRedisTokenBucketStore(s *RedisTokenBucketStore) {
	c := s.pool.Get()
	defer c.Close()
	c.Do("FLUSHDB")
}

func TestRedisTokenBucketStore(t *testing.T) {
	redisURL := os.Getenv("REDISTEST")
	if redisURL == "" {
		redisURL = "redis://127.0.0.1:6379/15"
	}
	store := NewRedisTokenBucketStore(redisURL, "", 3*time.Second, nil)
	defer clearRedisTokenBucketStore(store)

	Convey("RedisTokenBucketStore get and set", t, func() {
		info1 := TokenBucketInfo{
			Tokens:          42,
			LastRequestedAt: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		So(store.Set("id", info1), ShouldBeNil)

		info2, ok, err := store.Get("id")
		So(err, ShouldBeNil)
		So(ok, ShouldBeTrue)
		So(info1.Tokens, ShouldEqual, info2.Tokens)
		So(info1.LastRequestedAt.UnixNano(), ShouldEqual, info2.LastRequestedAt.UnixNano())

		time.Sleep(3 * time.Second)

		_, ok, err = store.Get("id")
		So(err, ShouldBeNil)
		So(ok, ShouldBeFalse)
	})
}
