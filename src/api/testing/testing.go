package testing

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/IWannaCommunity/gate-jump/src/api/database"
	"github.com/IWannaCommunity/gate-jump/src/api/log"
	"github.com/gorilla/mux"
)

// payload response returned from api. this should be passed in as an argument eventually
type Response struct {
	Success  bool        `json:"success"`
	Error    *string     `json:"error,omitempty"`
	Token    *string     `json:"token,omitempty"`
	User     interface{} `json:"user,omitempty"`
	UserList interface{} `json:"userList,omitempty"`
}

// test payload containing information about the request. should include response time
type TestPayload struct {
	Code     int
	Err      error
	Response *Response
}

// includes everything relevant to making requests to the api directly through the router
type TestingEnv struct {
	s           *sql.DB
	r           *mux.Router
	lastRequest interface{}
	method      string
	url         string
}

func (te *TestingEnv) ExpectedPayload(code int, err error, token string, user interface{}, userList interface{}) {

}

// should initalize the database on its own eventually by being passed in a string
func (te *TestingEnv) Init(s *sql.DB, r *mux.Router) {
	te.s = s
	te.r = r
}

func (te *TestingEnv) Prepare(method string, url string) {
	// clean database for new setup
	clearTable(te.s)

	// set method and url for api requests
	if method != "" {
		te.method = method
	}
	if url != "" {
		te.url = url
	}
}

func clearTable(db *sql.DB) {
	var err error
	_, err = db.Exec("DROP DATABASE gatejump")
	if err != nil {
		log.Fatal("1", err)
	}
	_, err = db.Exec("CREATE DATABASE gatejump")
	if err != nil {
		log.Fatal("2", err)
	}
	err = database.Init()
	if err != nil {
		log.Fatal("3", err)
	}
}

func ensureTableExists(db *sql.DB, creationQuery string) error {
	if _, err := db.Exec(creationQuery); err != nil {
		return err // do we care?
	}
	return nil
}

func (te *TestingEnv) Request(jsonRequest []byte) TestPayload {
	// Make API Request
	te.lastRequest = jsonRequest
	httpRequest, _ := http.NewRequest(te.method, te.url, bytes.NewBuffer(jsonRequest))
	httpTestRecorder := httptest.NewRecorder()
	te.r.ServeHTTP(httpTestRecorder, httpRequest)
	tp := TestPayload{}
	tp.Err = nil
	tp.Code = httpTestRecorder.Code
	temp, _ := ioutil.ReadAll(httpTestRecorder.Body)
	err := json.Unmarshal(temp, &tp.Response)
	if err != nil { // unmarshal failed somehow
		tp.Err = err
	}
	return tp
}

func (te *TestingEnv) Expect() string {
	return fmt.Sprintf("%s @ %s | \"%s\"", te.method, te.url, te.lastRequest)
}
