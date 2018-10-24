package main

import (
	"context"
	"database/sql"
	"fmt"
	"gate-jump/res"
	"log"
	"net/http"
	"os"

	jwt "github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type AuthLevel int

const (
	CLAIMS    AuthLevel = -1 // just the claims context tag
	PUBLIC    AuthLevel = 0  // public return
	USER      AuthLevel = 1  // user return
	ADMINUSER AuthLevel = 2  // admin wants to change password. admins cant change other users passwords so this exists
	ADMIN     AuthLevel = 3  // admin return
	SERVER    AuthLevel = 4  // server can update any user without giving 2 shits
)

type Server struct {
	Router  *mux.Router
	DB      *sql.DB
	LogFile *os.File
}

func (s *Server) Initialize(user, password, dbname string) {
	var err error
	s.DB, err = sql.Open("mysql", fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&interpolateParams=true", user, password, dbname))
	if err != nil {
		log.Fatal(err)
	}

	s.Router = mux.NewRouter()
}

func (s *Server) InitializeRoutes() {
	s.Router.HandleFunc("/", s.getAlive).Methods("GET")
	s.Router.HandleFunc("/user", s.getUsers).Methods("GET")
	s.Router.HandleFunc("/register", s.createUser).Methods("POST")
	s.Router.HandleFunc("/login", s.validateUser).Methods("POST")
	s.Router.HandleFunc("/user/{id}", s.getUser).Methods("GET")
	s.Router.HandleFunc("/user/{id}", s.updateUser).Methods("PUT")
	s.Router.HandleFunc("/user/{id}", s.deleteUser).Methods("DELETE")
	s.Router.Use(s.Recovery)
	s.Router.Use(s.JWTContext)

}

func (s *Server) Run(httpPort, httpsPort string) {
	if Config.Https.CertFile != "" && Config.Https.KeyFile != "" {
		log.Println("HTTPS Credentials picked up, running HTTPS")
		log.Fatal(http.ListenAndServeTLS(":"+httpsPort, Config.Https.CertFile, Config.Https.KeyFile, handlers.LoggingHandler(s.LogFile, s.Router)))
	} else {
		log.Println("HTTPS Credentials missing, running HTTP")
		log.Fatal(http.ListenAndServe(":"+httpPort, handlers.LoggingHandler(s.LogFile, s.Router)))
	}
}

func (s *Server) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				res.New(http.StatusInternalServerError).SetErrorMessage(fmt.Sprintf("%v", r)).Error(w)
				return
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type Claims struct {
	ID       int64   `json:"id"`
	Name     *string `json:"username"`
	Admin    bool    `json:"admin"`
	Country  *string `json:"country"`
	Locale   *string `json:"locale"`
	Verified bool    `json:"verified"`
	Banned   bool    `json:"banned"`
	jwt.StandardClaims
}

func (s *Server) JWTContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := Claims{}
		tokenString := r.Header.Get("Authorization")

		if tokenString == "" { // no token provided. public credential only
			ctx := context.WithValue(r.Context(), CLAIMS, Claims{ID: 0})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		// parse token provided
		token, err := jwt.ParseWithClaims(tokenString, &claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(Config.JwtSecret), nil
			})
		if err != nil { // token couldn't be read
			res.New(http.StatusUnauthorized).SetErrorMessage("Invalid Token Provided").Error(w)
			return
		}
		if !token.Valid { // token has been edited
			log.Println("hi?")
			res.New(http.StatusUnauthorized).SetErrorMessage("Token Invalid").Error(w)
			return
		}

		if token.Claims == nil {
			res.New(http.StatusInternalServerError).SetErrorMessage("Token Is Null").Error(w)
			return
		}
		claims = *token.Claims.(*Claims)

		ctx := context.WithValue(r.Context(), CLAIMS, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
