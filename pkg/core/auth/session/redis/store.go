package redis

import (
	"context"
	"encoding/json"
	"sort"
	gotime "time"

	goredis "github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/redis"
	"github.com/skygeario/skygear-server/pkg/core/time"
)

type StoreImpl struct {
	ctx         context.Context
	appID       string
	time        time.Provider
	logger      *logrus.Entry
	keyFunc     SessionKeyFunc
	listKeyFunc SessionListKeyFunc
}

var _ session.Store = &StoreImpl{}

func NewStore(
	ctx context.Context,
	appID string,
	time time.Provider,
	loggerFactory logging.Factory,
	keyFunc SessionKeyFunc,
	listKeyFunc SessionListKeyFunc,
) *StoreImpl {
	return &StoreImpl{
		ctx:         ctx,
		appID:       appID,
		time:        time,
		logger:      loggerFactory.NewLogger("redis-session-store"),
		keyFunc:     keyFunc,
		listKeyFunc: listKeyFunc,
	}
}

func (s *StoreImpl) Create(sess *auth.Session, expireAt gotime.Time) (err error) {
	json, err := json.Marshal(sess)
	if err != nil {
		return
	}
	expiry, err := expireAt.MarshalText()
	if err != nil {
		return
	}

	conn := redis.GetConn(s.ctx)
	ttl := expireAt.Sub(s.time.NowUTC())
	listKey := s.listKeyFunc(s.appID, sess.UserID)
	key := s.keyFunc(s.appID, sess.ID)

	_, err = conn.Do("HSET", listKey, key, expiry)
	if err != nil {
		err = errors.Newf("failed to update session list: %w", err)
		return
	}

	_, err = goredis.String(conn.Do("SET", key, json, "PX", toMilliseconds(ttl), "NX"))
	if errors.Is(err, goredis.ErrNil) {
		err = errors.Newf("duplicated session ID: %w", err)
		return
	}

	return
}

func (s *StoreImpl) Update(sess *auth.Session, expireAt gotime.Time) (err error) {
	data, err := json.Marshal(sess)
	if err != nil {
		return
	}
	expiry, err := expireAt.MarshalText()
	if err != nil {
		return
	}

	conn := redis.GetConn(s.ctx)
	ttl := expireAt.Sub(s.time.NowUTC())
	listKey := s.listKeyFunc(s.appID, sess.UserID)
	key := s.keyFunc(s.appID, sess.ID)

	_, err = conn.Do("HSET", listKey, key, expiry)
	if err != nil {
		err = errors.Newf("failed to update session list: %w", err)
		return
	}

	_, err = goredis.String(conn.Do("SET", key, data, "PX", toMilliseconds(ttl), "XX"))
	if errors.Is(err, goredis.ErrNil) {
		err = session.ErrSessionNotFound
	}
	return
}

func (s *StoreImpl) Get(id string) (sess *auth.Session, err error) {
	conn := redis.GetConn(s.ctx)
	key := s.keyFunc(s.appID, id)
	data, err := goredis.Bytes(conn.Do("GET", key))
	if errors.Is(err, goredis.ErrNil) {
		err = session.ErrSessionNotFound
		return
	} else if err != nil {
		return
	}
	err = json.Unmarshal(data, &sess)
	return
}

func (s *StoreImpl) Delete(session *auth.Session) (err error) {
	conn := redis.GetConn(s.ctx)
	key := s.keyFunc(s.appID, session.ID)
	listKey := s.listKeyFunc(s.appID, session.UserID)

	_, err = conn.Do("DEL", key)
	if err == nil {
		_, err = conn.Do("HDEL", listKey, key)
		if err != nil {
			s.logger.
				WithError(err).
				WithField("redis_key", listKey).
				Error("failed to update session list")
			// ignore non-critical errors
			err = nil
		}
	}
	return
}

func (s *StoreImpl) DeleteBatch(sessions []*auth.Session) (err error) {
	conn := redis.GetConn(s.ctx)

	sessionKeys := []interface{}{}
	listKeys := map[string]struct{}{}
	for _, session := range sessions {
		sessionKeys = append(sessionKeys, s.keyFunc(s.appID, session.ID))
		listKeys[s.listKeyFunc(s.appID, session.UserID)] = struct{}{}
	}
	if len(sessionKeys) == 0 {
		return nil
	}

	_, err = conn.Do("DEL", sessionKeys...)

	if err == nil {
		for listKey := range listKeys {
			_, err = conn.Do("HDEL", append([]interface{}{listKey}, sessionKeys...))
			if err != nil {
				s.logger.
					WithError(err).
					WithField("key", listKey).
					Error("failed to update session list")
				// ignore non-critical errors
				err = nil
			}
		}
	}
	return
}

func (s *StoreImpl) DeleteAll(userID string, sessionID string) error {
	conn := redis.GetConn(s.ctx)
	listKey := s.listKeyFunc(s.appID, userID)

	sessionKeys, err := goredis.Strings(conn.Do("HKEYS", listKey))
	if err != nil {
		return err
	}

	sessionKeysDel := []interface{}{}
	excludeSessionkey := s.keyFunc(s.appID, sessionID)
	for _, sessionKey := range sessionKeys {
		if excludeSessionkey == sessionKey {
			continue
		}
		sessionKeysDel = append(sessionKeysDel, sessionKey)
	}
	if len(sessionKeysDel) == 0 {
		return nil
	}

	_, err = conn.Do("DEL", sessionKeysDel...)
	if err == nil {
		args := append([]interface{}{listKey}, sessionKeysDel...)
		_, err = conn.Do("HDEL", args...)
		if err != nil {
			// ignore non-critical errors
			s.logger.
				WithError(err).
				WithField("key", listKey).
				Error("failed to update session list")
		}
	}

	return nil
}

func (s *StoreImpl) List(userID string) (sessions []*auth.Session, err error) {
	now := s.time.NowUTC()
	conn := redis.GetConn(s.ctx)
	listKey := s.listKeyFunc(s.appID, userID)

	sessionList, err := goredis.StringMap(conn.Do("HGETALL", listKey))
	if err != nil {
		return
	}

	for key, expiry := range sessionList {
		expireAt := gotime.Time{}
		err = expireAt.UnmarshalText([]byte(expiry))
		var expired bool
		if err != nil {
			s.logger.
				WithError(err).
				WithFields(logrus.Fields{"key": key, "expiry": expiry}).
				Error("invalid expiry value")
			err = nil
			// treat invalid value as expired
			expired = true
		} else {
			expired = now.After(expireAt)
		}

		session := &auth.Session{}
		var sessionJSON []byte
		sessionJSON, err = goredis.Bytes(conn.Do("GET", key))
		// key not found / invalid session JSON -> session not found
		if err == goredis.ErrNil {
			err = nil
			session = nil
		} else if err != nil {
			// unexpected error
			return
		} else {
			err = json.Unmarshal(sessionJSON, session)
			if err != nil {
				s.logger.
					WithError(err).
					WithFields(logrus.Fields{"key": key}).
					Error("invalid JSON value")
				err = nil
				session = nil
			}
		}

		if session == nil {
			// only cleanup expired sessions from the list
			if expired {
				// ignore non-critical error
				_, err = conn.Do("HDEL", listKey, key)
				if err != nil {
					// ignore non-critical error
					s.logger.
						WithError(err).
						WithFields(logrus.Fields{"key": listKey}).
						Error("failed to update session list")
					err = nil
				}
			}
		} else {
			sessions = append(sessions, session)
		}
	}

	sort.Sort(sessionSlice(sessions))
	return
}

func toMilliseconds(d gotime.Duration) int64 {
	return int64(d / gotime.Millisecond)
}

type sessionSlice []*auth.Session

func (s sessionSlice) Len() int           { return len(s) }
func (s sessionSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sessionSlice) Less(i, j int) bool { return s[i].CreatedAt.Before(s[j].CreatedAt) }
