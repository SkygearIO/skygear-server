package sso

import (
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
)

// CustomClaims is the type for jwt encoded
type CustomClaims struct {
	State
	jwt.StandardClaims
}

// NewState constructs a new state
func NewState(params GetURLParams) State {
	return params.State
}

// EncodeState encodes state by JWT
func EncodeState(secret string, state State) (string, error) {
	claims := CustomClaims{
		state,
		jwt.StandardClaims{},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// DecodeState decodes state by JWT
func DecodeState(secret string, encoded string) (*State, error) {
	claims := CustomClaims{}
	_, err := jwt.ParseWithClaims(encoded, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("fails to parse token")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return &claims.State, nil
}