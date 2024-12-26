package entities

import "github.com/golang-jwt/jwt/v4"

type AccessClaim struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaim struct {
	ID int `json:"id"`
	jwt.RegisteredClaims
}
