package oauth

import (
	"reflect"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
)

type MockProvider struct {
	Principals []*Principal
}

// NewMockProvider creates a new instance of mock provider
func NewMockProvider(principals []*Principal) *MockProvider {
	return &MockProvider{
		Principals: principals,
	}
}

func (m *MockProvider) GetPrincipalByProvider(options GetByProviderOptions) (*Principal, error) {
	if options.ProviderKeys == nil {
		options.ProviderKeys = map[string]interface{}{}
	}
	for _, p := range m.Principals {
		if p.ProviderType == options.ProviderType &&
			reflect.DeepEqual(p.ProviderKeys, options.ProviderKeys) &&
			p.ProviderUserID == options.ProviderUserID {
			return p, nil
		}
	}
	return nil, principal.ErrNotFound
}

func (m *MockProvider) GetPrincipalByUser(options GetByUserOptions) (*Principal, error) {
	if options.ProviderKeys == nil {
		options.ProviderKeys = map[string]interface{}{}
	}
	for _, p := range m.Principals {
		if p.ProviderType == options.ProviderType &&
			reflect.DeepEqual(p.ProviderKeys, options.ProviderKeys) &&
			p.UserID == options.UserID {
			return p, nil
		}
	}
	return nil, principal.ErrNotFound
}

func (m *MockProvider) CreatePrincipal(principal *Principal) error {
	m.Principals = append(m.Principals, principal)
	return nil
}

func (m *MockProvider) UpdatePrincipal(principal *Principal) error {
	for i, p := range m.Principals {
		if p.ID == principal.ID {
			m.Principals[i] = principal
		}
	}
	return nil
}

func (m *MockProvider) DeletePrincipal(principal *Principal) error {
	j := -1
	for i, p := range m.Principals {
		if p.ID == principal.ID {
			j = i
			break
		}
	}
	if j != -1 {
		m.Principals = append(m.Principals[:j], m.Principals[j+1:]...)
	}
	return nil
}

func (m *MockProvider) GetPrincipalsByUserID(userID string) ([]*Principal, error) {
	var principals []*Principal
	for _, p := range m.Principals {
		if p.UserID == userID {
			principal := p
			principals = append(principals, principal)
		}
	}
	return principals, nil
}

func (m *MockProvider) GetPrincipalsByClaim(claimName string, claimValue string) ([]*Principal, error) {
	var principals []*Principal
	for _, p := range m.Principals {
		if p.ClaimsValue[claimName] == claimValue {
			principal := p
			principals = append(principals, principal)
		}
	}
	return principals, nil
}

func (m *MockProvider) ID() string {
	return string(coreAuth.PrincipalTypeOAuth)
}

func (m *MockProvider) GetPrincipalByID(id string) (principal.Principal, error) {
	for _, p := range m.Principals {
		if p.ID == id {
			principal := p
			return principal, nil
		}
	}
	return nil, principal.ErrNotFound
}

func (m *MockProvider) ListPrincipalsByUserID(userID string) ([]principal.Principal, error) {
	principals, err := m.GetPrincipalsByUserID(userID)
	if err != nil {
		return nil, err
	}

	genericPrincipals := []principal.Principal{}
	for _, principal := range principals {
		genericPrincipals = append(genericPrincipals, principal)
	}

	return genericPrincipals, nil
}

func (m *MockProvider) ListPrincipalsByClaim(claimName string, claimValue string) ([]principal.Principal, error) {
	principals, err := m.GetPrincipalsByClaim(claimName, claimValue)
	if err != nil {
		return nil, err
	}

	genericPrincipals := []principal.Principal{}
	for _, principal := range principals {
		genericPrincipals = append(genericPrincipals, principal)
	}

	return genericPrincipals, nil
}

var (
	_ Provider = &MockProvider{}
)
