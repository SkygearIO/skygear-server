package hook

import (
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"
	"github.com/skygeario/skygear-server/pkg/auth/event"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal/password"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/config"
)

type mutatorImpl struct {
	Event                  *event.Event
	User                   *model.User
	UserPasswordPrincipals *[]*password.Principal
	Mutations              event.Mutations

	UserVerificationConfig *config.UserVerificationConfiguration
	PasswordAuthProvider   password.Provider
	AuthInfoStore          authinfo.Store
	UserProfileStore       userprofile.Store
}

func NewMutator(
	verifyConfig *config.UserVerificationConfiguration,
	passwordProvider password.Provider,
	authInfoStore authinfo.Store,
	userProfileStore userprofile.Store,
) Mutator {
	return &mutatorImpl{
		UserVerificationConfig: verifyConfig,
		PasswordAuthProvider:   passwordProvider,
		AuthInfoStore:          authInfoStore,
		UserProfileStore:       userProfileStore,
	}
}

func (mutator *mutatorImpl) New(ev *event.Event, user *model.User) Mutator {
	newMutator := *mutator
	newMutator.Event = ev
	newMutator.User = user
	newMutator.Mutations = event.Mutations{}
	return &newMutator
}

func (mutator *mutatorImpl) Add(mutations event.Mutations) error {
	// update computed verified status if needed
	if mutations.VerifyInfo != nil || mutations.IsManuallyVerified != nil {
		// update IsVerified
		if mutator.UserPasswordPrincipals == nil {
			principals, err := mutator.PasswordAuthProvider.GetPrincipalsByUserID(mutator.User.ID)
			if err != nil {
				return err
			}
			mutator.UserPasswordPrincipals = &principals
		}

		verifyInfo := mutator.User.VerifyInfo
		if mutations.VerifyInfo != nil {
			verifyInfo = *mutations.VerifyInfo
		}
		isVerified := userverify.IsUserVerified(
			verifyInfo,
			*mutator.UserPasswordPrincipals,
			mutator.UserVerificationConfig.Criteria,
			mutator.UserVerificationConfig.LoginIDKeys,
		)
		mutations.IsComputedVerified = &isVerified
	}

	mutator.Mutations = mutator.Mutations.WithMutationsApplied(mutations)
	if payload, ok := mutator.Event.Payload.(event.UserAwarePayload); ok {
		mutator.Event.Payload = payload.WithMutationsApplied(mutator.Mutations)
	}
	mutator.Mutations.ApplyToUser(mutator.User)
	return nil
}

func (mutator *mutatorImpl) Apply() error {
	mutations := mutator.Mutations

	// mutate user profile
	if mutations.IsNoop() {
		return nil
	}

	if mutations.Metadata != nil {
		_, err := mutator.UserProfileStore.UpdateUserProfile(mutator.User.ID, *mutations.Metadata)
		if err != nil {
			return err
		}
		mutations.Metadata = nil
	}

	// mutate auth info
	if mutations.IsNoop() {
		return nil
	}

	var authInfo authinfo.AuthInfo
	err := mutator.AuthInfoStore.GetAuth(mutator.User.ID, &authInfo)
	if err != nil {
		return err
	}
	if mutations.IsDisabled != nil {
		authInfo.Disabled = *mutations.IsDisabled
		authInfo.DisabledMessage = ""
		authInfo.DisabledExpiry = nil // never expire
	}
	if mutations.IsManuallyVerified != nil {
		authInfo.ManuallyVerified = *mutations.IsManuallyVerified
	}
	if mutations.VerifyInfo != nil {
		authInfo.VerifyInfo = *mutations.VerifyInfo
	}

	if mutations.VerifyInfo != nil {
		isVerified := userverify.IsUserVerified(
			authInfo.VerifyInfo,
			*mutator.UserPasswordPrincipals,
			mutator.UserVerificationConfig.Criteria,
			mutator.UserVerificationConfig.LoginIDKeys,
		)
		authInfo.Verified = isVerified
	}

	err = mutator.AuthInfoStore.UpdateAuth(&authInfo)
	if err != nil {
		return err
	}

	return nil
}
