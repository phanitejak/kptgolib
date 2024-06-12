package httpmiddleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go/v4"
)

// GetUserNameFromRequestTokenOrElse returns user name from Authorization token or elseName if no such exists.
func GetUserNameFromRequestTokenOrElse(r *http.Request, elseName string) string {
	u, err := GetUserNameFromRequestToken(r)
	if err != nil {
		return elseName
	}
	return u
}

// GetUserNameFromRequestToken retrieves the userName from the http Authorization header.
func GetUserNameFromRequestToken(r *http.Request) (string, error) {
	tokenString := tokenFromHeader(r)
	if tokenString == "" {
		return "", fmt.Errorf("authorization header invalid or not found")
	}

	parser := jwt.Parser{}

	type UserNameClaims struct {
		UserName string `json:"preferred_username"`
		jwt.Claims
	}

	claims := UserNameClaims{}

	token, _, err := parser.ParseUnverified(tokenString, &claims)
	if err != nil {
		return "", err
	}

	userName := token.Claims.(*UserNameClaims).UserName
	if userName == "" {
		return userName, fmt.Errorf("field 'preferred_username' not found or empty in authorization token")
	}

	return userName, nil
}

func tokenFromHeader(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}
	return ""
}
