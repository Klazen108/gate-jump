package authentication

import (
	"context"
	"net/http"

	"github.com/IWannaCommunity/gate-jump/src/api/res"
	"github.com/IWannaCommunity/gate-jump/src/api/settings"
	jwt "github.com/dgrijalva/jwt-go"
)

type ContextKey int
type AuthLevel int

// Auth Levels
const (
	PUBLIC    AuthLevel = 0 // public return
	USER      AuthLevel = 1 // user return
	ADMINUSER AuthLevel = 2 // admin wants to change password. admins cant change other users passwords so this exists
	ADMIN     AuthLevel = 3 // admin return
	SERVER    AuthLevel = 4 // server has mega powerlevel
)

// Context Keys
const (
	ClaimsKey ContextKey = -1 // claims context tag
)

type Context struct {
	Claims Claims
	Token  string
}

type Claims struct {
	UUID    *string  `json:"uuid"`
	Name    *string  `json:"username"`
	Country *string  `json:"country"`
	Locale  *string  `json:"locale"`
	Group   []string `json:"group"`
	Scope   []string `json:"scope"`
	jwt.StandardClaims
}

func JWTContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := Context{}
		tokenString := r.Header.Get("Authorization")

		if tokenString == "" { // no token provided, value checked will be nil
			ctx := context.WithValue(r.Context(), ClaimsKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// parse token provided
		token, err := jwt.ParseWithClaims(tokenString, &claims.Claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(settings.JwtSecret), nil
			})
		if err != nil { // token couldn't be read
			res.New(http.StatusUnauthorized).SetErrorMessage("Invalid Token Provided").Error(w)
			return
		}
		if !token.Valid { // token has been edited
			res.New(http.StatusUnauthorized).SetErrorMessage("Token Is Invalid").Error(w)
			return
		}
		if token.Claims == nil { // nothing was put into the token
			res.New(http.StatusInternalServerError).SetErrorMessage("Token Is Null").Error(w)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
