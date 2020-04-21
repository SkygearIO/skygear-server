package forgotpassword

import (
	"time"

	"github.com/skygeario/skygear-server/pkg/core/base32"
	"github.com/skygeario/skygear-server/pkg/core/crypto"
	"github.com/skygeario/skygear-server/pkg/core/rand"
)

type Code struct {
	CodeHash    string    `json:"code_hash"`
	PrincipalID string    `json:"principal_id"`
	CreatedAt   time.Time `json:"created_at"`
	ExpireAt    time.Time `json:"expire_at"`
	Consumed    bool      `json:"consumed"`
}

func GenerateCode() string {
	code := rand.StringWithAlphabet(32, base32.Alphabet, rand.SecureRand)
	return code
}

func HashCode(code string) string {
	return crypto.SHA256String(code)
}
