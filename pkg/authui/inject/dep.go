package inject

import (
	"github.com/gomodule/redigo/redis"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
)

type EnableFileSystemTemplate bool

type BootTimeDependency struct {
	Configuration                 Configuration
	DBPool                        db.Pool
	RedisPool                     *redis.Pool
	StandaloneTenantConfiguration *config.TenantConfiguration
}
