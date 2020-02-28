package authnsession

import (
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/mfa"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	authTesting "github.com/skygeario/skygear-server/pkg/core/auth/testing"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/time"
)

func NewMockProvider(
	mfaConfiguration *config.MFAConfiguration,
	timeProvider time.Provider,
	mfaProvider mfa.Provider,
	authInfoStore authinfo.Store,
	sessionProvider session.Provider,
	sessionWriter session.Writer,
	identityProvider principal.IdentityProvider,
	hookProvider hook.Provider,
	userProfileStore userprofile.Store,
) Provider {
	authContext := authTesting.NewMockContext()
	authenticationSessionConfiguration :=
		&config.AuthenticationSessionConfiguration{
			Secret: "authnsessionsecret",
		}
	return NewProvider(
		authContext,
		mfaConfiguration,
		authenticationSessionConfiguration,
		timeProvider,
		mfaProvider,
		authInfoStore,
		sessionProvider,
		sessionWriter,
		identityProvider,
		hookProvider,
		userProfileStore,
	)
}
