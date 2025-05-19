package server

import (
	"errors"

	jwt "github.com/golang-jwt/jwt/v5"
)

func generateJWTToken(signingKey string, claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(signingKey))
}

func parseJWTToken(signingKey, tokenString string, outClaims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, outClaims, func(token *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return err
	}
	if !token.Valid {
		return errors.New("token is invalid")
	}
	return nil
}
