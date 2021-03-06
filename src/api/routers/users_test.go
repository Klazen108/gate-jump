package routers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/IWannaCommunity/gate-jump/src/api/mailer"
	"github.com/IWannaCommunity/gate-jump/src/api/settings"

	"github.com/IWannaCommunity/gate-jump/src/api/database"
	"github.com/IWannaCommunity/gate-jump/src/api/log"
	tst "github.com/IWannaCommunity/gate-jump/src/api/testing"
	"github.com/stretchr/testify/assert"
)

var te *tst.TestingEnv

func TestMain(m *testing.M) {
	var err error

	err = settings.FromFile("../config/config.json")
	if err != nil {
		log.Fatal(err) // clearly couldn't get database variables
	}

	go database.Connect("root", "", "gatejump") // connect the database for the database package
	go Serve("10421", "444")                    // run router on port
	err = mailer.SMTPInit()
	if err != nil {
		log.Fatal(err)
	}
	go mailer.Daemon()

	testdb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&interpolateParams=true", "root", "", "gatejump"))
	if err != nil {
		log.Fatal(err) // can't run tests if we can't initialize the database
	}

	for !database.Initialized() { // checking that database package database object is initalized
	}
	for router == nil { // checking that router package router object is initalized
	}
	for mailer.Outbox == nil { // checking that mailer object is initalized
	}

	te = &tst.TestingEnv{}
	te.Init(testdb, router)

	code := m.Run() // run tests

	os.Exit(code) // we finished the tsts

}

func TestAlive(t *testing.T) {
	te.Prepare("GET", "/")
	r := te.Request(nil)

	if assert.NoError(t, r.Err, te.Expect()) {

		assert.Equal(t, http.StatusOK, r.Code, te.Expect())
		assert.True(t, r.Response.Success, te.Expect())

		assert.Nil(t, r.Response.Error, te.Expect())
		assert.Nil(t, r.Response.Token, te.Expect())
		assert.Nil(t, r.Response.User, te.Expect())
		assert.Nil(t, r.Response.UserList, te.Expect())

	}

}

func TestCreateUser(t *testing.T) {
	te.Prepare("POST", "/register")

	var badRequests []string // only valid request should be one that contains name, email, and password

	badRequests = append(badRequests,
		`sdfdrslkjgnm4momgom!!!`,                                                             // jibberish
		`{"password":"12345678","email":"email@website.com"}`,                                // missing name
		`{"name":"test_user","email":"email@website.com"}`,                                   // missing password
		`{"name":"test_user","password":"12345678"}`,                                         // missing email
		`{"name":"test_user","password":"12345678","country":"us","locale":"en"}`,            // extra
		`{"name":"12356","password":"12345678","email":"email@website.com"}`,                 // invalid username (all numerics)
		`{"name":"test_user@website.com","password":"12345678","email":"email@website.com"}`, // invalid username (its an email)
		`{"name":"test_user","password":"12345","email":"email@website.com"}`,                // invalid password (less than 8 characters)
		`{"name":"test_user","password":"12345678","email":"email"}`)                         // invalid email (non-email format)
	mainUser := `{"name":"test_user","password":"12345678","email":"email@website.com"}`               // valid request
	duplicateName := `{"name":"test_user","password":"12345678","email":"email@someotherwebsite.com"}` // name == mainUser.Name
	duplicateEmail := `{"name":"some_other_user","password":"12345678","email":"email@website.com"}`   // email == mainUser.Email

	// test bad request
	for i, badRequest := range badRequests {
		r := te.Request([]byte(badRequest))
		if assert.NoError(t, r.Err, te.Expect()) {

			assert.Equal(t, http.StatusBadRequest, r.Code, te.Expect())
			assert.False(t, r.Response.Success, te.Expect())
			if assert.NotNil(t, r.Response.Error, te.Expect()) {
				switch i {
				case 5:
					fallthrough
				case 6: // invalid username
					assert.Equal(t, "Invalid Username Format", *r.Response.Error, te.Expect())
				case 7:
					assert.Equal(t, "Invalid Password Format", *r.Response.Error, te.Expect())
				case 8:
					assert.Equal(t, "Invalid Email Format", *r.Response.Error, te.Expect())
				default:
					assert.Equal(t, "Invalid Request Payload", *r.Response.Error, te.Expect())
				}
			}

			assert.Nil(t, r.Response.Token, te.Expect())
			assert.Nil(t, r.Response.User, te.Expect())
			assert.Nil(t, r.Response.UserList, te.Expect())
		}
	}

	// test creating a user
	r := te.Request([]byte(mainUser))
	if assert.NoError(t, r.Err) {

		assert.Equal(t, http.StatusCreated, r.Code, te.Expect())
		assert.True(t, r.Response.Success, te.Expect())

		assert.Nil(t, r.Response.Error, te.Expect())
		assert.Nil(t, r.Response.Token, te.Expect())
		assert.Nil(t, r.Response.User, te.Expect())
		assert.Nil(t, r.Response.UserList, te.Expect())
	}

	// test duplicate username
	r = te.Request([]byte(duplicateName))
	if assert.NoError(t, r.Err) {

		assert.Equal(t, http.StatusConflict, r.Code, te.Expect())
		assert.False(t, r.Response.Success, te.Expect())
		if assert.NotNil(t, r.Response.Error) {
			assert.Equal(t, "Username Already Exists", *r.Response.Error, te.Expect())
		}

		assert.Nil(t, r.Response.Token)
		assert.Nil(t, r.Response.User)
		assert.Nil(t, r.Response.UserList)
	}

	// test duplicate email
	r = te.Request([]byte(duplicateEmail))
	if assert.NoError(t, r.Err) {

		assert.Equal(t, http.StatusConflict, r.Code, "expected statusconflict")
		assert.False(t, r.Response.Success)
		if assert.NotNil(t, r.Response.Error) {
			assert.Equal(t, "Email Already In Use", *r.Response.Error)
		}

		assert.Nil(t, r.Response.Token)
		assert.Nil(t, r.Response.User)
		assert.Nil(t, r.Response.UserList)
	}
}
