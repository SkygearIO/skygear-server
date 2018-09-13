package provider

import (
	"context"

	"github.com/skygeario/skygear-server/pkg/auth/db"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authtoken"
	"github.com/skygeario/skygear-server/pkg/core/config"
	coreDB "github.com/skygeario/skygear-server/pkg/core/db"
)

type AuthProviders struct {
	DB            *coreDB.DBProvider
	TokenStore    *authtoken.StoreProvider
	AuthInfoStore *authinfo.StoreProvider
}

func (d AuthProviders) Provide(dependencyName string, ctx context.Context, tConfig config.TenantConfiguration) interface{} {
	switch dependencyName {
	case "DB":
		return d.ProvideDB(ctx, tConfig)
	case "TokenStore":
		return d.TokenStore.Provide(ctx, tConfig)
	case "AuthInfoStore":
		return d.AuthInfoStore.Provide(ctx, tConfig)
	default:
		return nil
	}
}

func (d AuthProviders) ProvideDB(ctx context.Context, tConfig config.TenantConfiguration) *db.DBConn {
	conn := d.DB.Provide(ctx, tConfig)
	db := &db.DBConn{
		conn,
		tConfig.DBConnectionStr,
	}
	return db
}
