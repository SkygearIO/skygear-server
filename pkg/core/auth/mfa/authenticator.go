package mfa

import (
	"time"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/mail"
	"github.com/skygeario/skygear-server/pkg/core/phone"
)

type Authenticator interface {
	GetID() string
	GetUserID() string
	GetType() coreAuth.AuthenticatorType
	GetActivated() bool
	GetCreatedAt() time.Time
	GetActivatedAt() *time.Time
}

type TOTPAuthenticator struct {
	ID          string
	UserID      string
	Type        coreAuth.AuthenticatorType
	Activated   bool
	CreatedAt   time.Time
	ActivatedAt *time.Time
	Secret      string
	DisplayName string
}

func (a TOTPAuthenticator) GetID() string {
	return a.ID
}

func (a TOTPAuthenticator) GetUserID() string {
	return a.UserID
}

func (a TOTPAuthenticator) GetType() coreAuth.AuthenticatorType {
	return a.Type
}

func (a TOTPAuthenticator) GetActivated() bool {
	return a.Activated
}

func (a TOTPAuthenticator) GetCreatedAt() time.Time {
	return a.CreatedAt
}

func (a TOTPAuthenticator) GetActivatedAt() *time.Time {
	return a.ActivatedAt
}

func (a TOTPAuthenticator) Mask() MaskedTOTPAuthenticator {
	return MaskedTOTPAuthenticator{
		ID:          a.ID,
		UserID:      a.UserID,
		Type:        a.Type,
		CreatedAt:   a.CreatedAt,
		Activated:   a.Activated,
		ActivatedAt: a.ActivatedAt,
		DisplayName: a.DisplayName,
	}
}

type MaskedTOTPAuthenticator struct {
	ID          string                     `json:"id"`
	UserID      string                     `json:"-"`
	Type        coreAuth.AuthenticatorType `json:"type"`
	CreatedAt   time.Time                  `json:"created_at"`
	Activated   bool                       `json:"-"`
	ActivatedAt *time.Time                 `json:"activated_at"`
	DisplayName string                     `json:"display_name"`
}

func (a MaskedTOTPAuthenticator) GetID() string {
	return a.ID
}

func (a MaskedTOTPAuthenticator) GetUserID() string {
	return a.UserID
}

func (a MaskedTOTPAuthenticator) GetType() coreAuth.AuthenticatorType {
	return a.Type
}

func (a MaskedTOTPAuthenticator) GetActivated() bool {
	return a.Activated
}

func (a MaskedTOTPAuthenticator) GetCreatedAt() time.Time {
	return a.CreatedAt
}

func (a MaskedTOTPAuthenticator) GetActivatedAt() *time.Time {
	return a.ActivatedAt
}

type OOBAuthenticator struct {
	ID          string
	UserID      string
	Type        coreAuth.AuthenticatorType
	Activated   bool
	CreatedAt   time.Time
	ActivatedAt *time.Time
	Channel     coreAuth.AuthenticatorOOBChannel
	Phone       string
	Email       string
}

func (a OOBAuthenticator) GetID() string {
	return a.ID
}

func (a OOBAuthenticator) GetUserID() string {
	return a.UserID
}

func (a OOBAuthenticator) GetType() coreAuth.AuthenticatorType {
	return a.Type
}

func (a OOBAuthenticator) GetActivated() bool {
	return a.Activated
}

func (a OOBAuthenticator) GetCreatedAt() time.Time {
	return a.CreatedAt
}

func (a OOBAuthenticator) GetActivatedAt() *time.Time {
	return a.ActivatedAt
}

func (a OOBAuthenticator) Mask() MaskedOOBAuthenticator {
	return MaskedOOBAuthenticator{
		ID:          a.ID,
		UserID:      a.UserID,
		Type:        a.Type,
		CreatedAt:   a.CreatedAt,
		Activated:   a.Activated,
		ActivatedAt: a.ActivatedAt,
		Channel:     a.Channel,
		MaskedPhone: phone.Mask(a.Phone),
		MaskedEmail: mail.MaskAddress(a.Email),
	}
}

type MaskedOOBAuthenticator struct {
	ID          string                           `json:"id"`
	UserID      string                           `json:"-"`
	Type        coreAuth.AuthenticatorType       `json:"type"`
	CreatedAt   time.Time                        `json:"created_at"`
	Activated   bool                             `json:"-"`
	ActivatedAt *time.Time                       `json:"activated_at"`
	Channel     coreAuth.AuthenticatorOOBChannel `json:"channel"`
	MaskedPhone string                           `json:"masked_phone,omitempty"`
	MaskedEmail string                           `json:"masked_email,omitempty"`
}

func (a MaskedOOBAuthenticator) GetID() string {
	return a.ID
}

func (a MaskedOOBAuthenticator) GetUserID() string {
	return a.UserID
}

func (a MaskedOOBAuthenticator) GetType() coreAuth.AuthenticatorType {
	return a.Type
}

func (a MaskedOOBAuthenticator) GetActivated() bool {
	return a.Activated
}

func (a MaskedOOBAuthenticator) GetCreatedAt() time.Time {
	return a.CreatedAt
}

func (a MaskedOOBAuthenticator) GetActivatedAt() *time.Time {
	return a.ActivatedAt
}

type RecoveryCodeAuthenticator struct {
	ID        string
	UserID    string
	Type      coreAuth.AuthenticatorType
	Code      string
	CreatedAt time.Time
	Consumed  bool
}

type BearerTokenAuthenticator struct {
	ID        string
	UserID    string
	Type      coreAuth.AuthenticatorType
	ParentID  string
	Token     string
	CreatedAt time.Time
	ExpireAt  time.Time
}

type OOBCode struct {
	ID              string
	UserID          string
	AuthenticatorID string
	Code            string
	CreatedAt       time.Time
	ExpireAt        time.Time
}

func MaskAuthenticators(authenticators []Authenticator) []Authenticator {
	output := make([]Authenticator, len(authenticators))
	for i, a := range authenticators {
		switch aa := a.(type) {
		case TOTPAuthenticator:
			output[i] = aa.Mask()
		case OOBAuthenticator:
			output[i] = aa.Mask()
		default:
			panic("mfa: unknown authenticator")
		}
	}
	return output
}

func CanAddAuthenticator(authenticators []Authenticator, newA Authenticator, mfaConfiguration *config.MFAConfiguration) bool {
	// Always return false if MFA is off.
	if !mfaConfiguration.Enabled {
		return false
	}

	// Calculate the count
	totalCount := len(authenticators)
	totpCount := 0
	oobSMSCount := 0
	oobEmailCount := 0
	incrFunc := func(a Authenticator) {
		switch aa := a.(type) {
		case TOTPAuthenticator:
			totpCount++
		case OOBAuthenticator:
			switch aa.Channel {
			case coreAuth.AuthenticatorOOBChannelSMS:
				oobSMSCount++
			case coreAuth.AuthenticatorOOBChannelEmail:
				oobEmailCount++
			default:
				panic("mfa: unknown OOB authenticator channel")
			}
		default:
			panic("mfa: unknown authenticator")
		}
	}

	for _, a := range authenticators {
		incrFunc(a)
	}

	// Simulate the count if new one is added.
	totalCount++
	incrFunc(newA)

	// Compare the count
	if totalCount > *mfaConfiguration.Maximum {
		return false
	}
	if totpCount > *mfaConfiguration.TOTP.Maximum {
		return false
	}
	if oobSMSCount > *mfaConfiguration.OOB.SMS.Maximum {
		return false
	}
	if oobEmailCount > *mfaConfiguration.OOB.Email.Maximum {
		return false
	}

	return true
}

// IsDeletingOnlyActivatedAuthenticator checks if authenticators is of length 1 and
// a is activated and a is in authenticators.
func IsDeletingOnlyActivatedAuthenticator(authenticators []Authenticator, a Authenticator) bool {
	id := a.GetID()
	activated := a.GetActivated()
	if !activated {
		return false
	}
	if len(authenticators) != 1 {
		return false
	}
	for _, aa := range authenticators {
		if aa.GetID() == id {
			return true
		}
	}
	return false
}

var (
	_ Authenticator = TOTPAuthenticator{}
	_ Authenticator = OOBAuthenticator{}
	_ Authenticator = MaskedTOTPAuthenticator{}
	_ Authenticator = MaskedOOBAuthenticator{}
)
