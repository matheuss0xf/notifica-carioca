package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/gen_jwt <cpf>")
		os.Exit(1)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-jwt-secret"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"preferred_username": os.Args[1],
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "signing token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signed)
}
