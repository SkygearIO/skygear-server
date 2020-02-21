package inject

import (
	"github.com/gomodule/redigo/redis"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type EnableFileSystemTemplate bool

type BootTimeDependency struct {
	Configuration                 Configuration
	DBPool                        db.Pool
	RedisPool                     *redis.Pool
	StandaloneTenantConfiguration *config.TenantConfiguration
	Validator                     *validation.Validator
	ReservedNameChecker           *loginid.ReservedNameChecker
}
