package password

import (
	"github.com/skygeario/skygear-server/pkg/core/auth/passwordpolicy"
)

type ResetPasswordRequestContext struct {
	PasswordChecker      *passwordpolicy.PasswordChecker
	PasswordAuthProvider Provider
}

func (r *ResetPasswordRequestContext) ExecuteWithPrincipals(newPassword string, principals []*Principal) (err error) {
	if err = r.PasswordChecker.ValidatePassword(passwordpolicy.ValidatePasswordPayload{
		PlainPassword: newPassword,
	}); err != nil {
		return
	}

	for _, p := range principals {
		err = r.PasswordAuthProvider.UpdatePassword(p, newPassword)
		if err != nil {
			return
		}
	}

	return
}

func (r *ResetPasswordRequestContext) ExecuteWithUserID(newPassword string, userID string) (err error) {
	principals, err := r.PasswordAuthProvider.GetPrincipalsByUserID(userID)
	if err != nil {
		return
	}

	err = r.ExecuteWithPrincipals(newPassword, principals)
	return
}
