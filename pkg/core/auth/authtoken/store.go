package authtoken

import (
	"github.com/skygeario/skygear-server/pkg/server/authtoken"
)

type Store interface {
	NewToken(appName string, authInfoID string) (authtoken.Token, error)
	Get(accessToken string, token *authtoken.Token) error
	Put(token *authtoken.Token) error
	Delete(accessToken string) error
}
