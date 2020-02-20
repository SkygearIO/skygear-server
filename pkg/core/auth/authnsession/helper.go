package authnsession

import (
	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/mfa"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
)

func GetRequiredSteps(mfaProvider mfa.Provider, mfaConfig *config.MFAConfiguration, userID string) ([]auth.AuthnSessionStep, error) {
	steps := []auth.AuthnSessionStep{auth.AuthnSessionStepIdentity}
	enforcement := mfaConfig.Enforcement
	switch enforcement {
	case config.MFAEnforcementOptional:
		authenticators, err := mfaProvider.ListAuthenticators(userID)
		if err != nil {
			return nil, err
		}
		if len(authenticators) > 0 {
			steps = append(steps, auth.AuthnSessionStepMFA)
		}
	case config.MFAEnforcementRequired:
		steps = append(steps, auth.AuthnSessionStepMFA)
	case config.MFAEnforcementOff:
		break
	default:
		return nil, errors.New("unknown MFA enforcement")
	}
	return steps, nil
}
