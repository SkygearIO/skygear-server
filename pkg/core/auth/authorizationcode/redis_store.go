package authorizationcode

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/gomodule/redigo/redis"

	"github.com/skygeario/skygear-server/pkg/core/base32"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/rand"
	"github.com/skygeario/skygear-server/pkg/core/redis"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"
)

type KeyFunc func(appID string, code string) string

type RedisStore struct {
	AppID        string
	Context      context.Context
	TimeProvider coreTime.Provider
	KeyFunc      KeyFunc
}

var _ Store = &RedisStore{}

func NewRedisStore(
	context context.Context,
	tConfig *config.TenantConfiguration,
	timeProvider coreTime.Provider,
	keyFunc KeyFunc,
) *RedisStore {
	return &RedisStore{
		AppID:        tConfig.AppID,
		Context:      context,
		TimeProvider: timeProvider,
		KeyFunc:      keyFunc,
	}
}

func (s *RedisStore) New(t *T) (code string, err error) {
	bytes, err := json.Marshal(t)
	if err != nil {
		return
	}
	code = rand.StringWithAlphabet(Length, base32.Alphabet, rand.SecureRand)
	conn := redis.GetConn(s.Context)
	key := s.KeyFunc(s.AppID, code)

	_, err = conn.Do("SET", key, bytes, "PX", coreTime.ToMilliseconds(5*time.Minute))
	if err != nil {
		return
	}

	return
}

func (s *RedisStore) Consume(code string) (t *T, err error) {
	conn := redis.GetConn(s.Context)
	key := s.KeyFunc(s.AppID, code)

	conn.Send("MULTI")
	conn.Send("GET", key)
	conn.Send("DEL", key)
	r, err := conn.Do("EXEC")
	if err != nil {
		return
	}

	bytes, err := goredis.Bytes(r.([]interface{})[0], err)
	if errors.Is(err, goredis.ErrNil) {
		err = ErrInvalidCode
	}
	if err != nil {
		return
	}

	t = &T{}
	err = json.Unmarshal(bytes, t)
	if err != nil {
		return
	}

	return
}
