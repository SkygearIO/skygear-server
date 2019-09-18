package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/FZambia/sentinel"
	"github.com/gomodule/redigo/redis"
)

// When sentinel is enabled, host and port will be ignored
type Configuration struct {
	SentinelEnabled bool           `envconfig:"SENTINEL_ENABLED"`
	Host            string         `envconfig:"HOST"`
	Port            int            `envconfig:"PORT" default:"6379"`
	Password        string         `envconfig:"PASSWORD"`
	DB              int            `envconfig:"DB" default:"0"`
	Sentinel        SentinelConfig `envconfig:"SENTINEL"`
}

type SentinelConfig struct {
	Addrs      []string `envconfig:"ADDRS"`
	MasterName string   `envconfig:"MASTER_NAME" default:"mymaster"`
}

func NewPool(config Configuration) *redis.Pool {
	if config.SentinelEnabled {
		return newSentinelPool(config)
	}
	hostPort := fmt.Sprintf("%s:%d", config.Host, config.Port)
	// TODO(pool): configurable / profile for good value?
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 5 * time.Minute,
		Dial: func() (conn redis.Conn, err error) {
			conn, err = redis.Dial(
				"tcp",
				hostPort,
				redis.DialDatabase(config.DB),
				redis.DialPassword(config.Password),
			)
			return
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) (err error) {
			_, err = conn.Do("PING")
			return
		},
	}
}

func newSentinelPool(config Configuration) *redis.Pool {
	sntnl := &sentinel.Sentinel{
		Addrs:      config.Sentinel.Addrs,
		MasterName: config.Sentinel.MasterName,
		Dial: func(addr string) (redis.Conn, error) {
			timeout := 500 * time.Millisecond
			c, err := redis.Dial(
				"tcp",
				addr,
				redis.DialConnectTimeout(timeout),
				redis.DialReadTimeout(timeout),
				redis.DialWriteTimeout(timeout),
			)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   64,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			masterAddr, err := sntnl.MasterAddr()
			if err != nil {
				return nil, err
			}
			c, err := redis.Dial(
				"tcp",
				masterAddr,
				redis.DialDatabase(config.DB),
				redis.DialPassword(config.Password),
			)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if !sentinel.TestRole(c, "master") {
				return errors.New("Role check failed")
			}
			return nil
		},
	}
}
