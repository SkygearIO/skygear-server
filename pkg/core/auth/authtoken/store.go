package authtoken

import (
	"context"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/server/authtoken"
)

type StoreProvider struct{}

func (p StoreProvider) Provide(ctx context.Context, tConfig config.TenantConfiguration) interface{} {
	return authtoken.NewJWTStore(tConfig.TokenStore.Secret, tConfig.TokenStore.Expiry)
}

type Store interface {
	NewToken(appName string, authInfoID string) (authtoken.Token, error)
	Get(accessToken string, token *authtoken.Token) error
	Put(token *authtoken.Token) error
	Delete(accessToken string) error
}
