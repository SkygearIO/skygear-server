package loginid

import (
	"github.com/skygeario/skygear-server/pkg/core/utils"
)

const DefaultRealm string = "default"

type RealmChecker interface {
	IsValid(realm string) bool
}

type DefaultRealmChecker struct {
	AllowedRealms []string
}

func (c *DefaultRealmChecker) IsValid(realm string) bool {
	return utils.StringSliceContains(c.AllowedRealms, realm)
}

var (
	_ RealmChecker = &DefaultRealmChecker{}
)
