/*
Package handler contains the methods used by the router
*/
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/GGP1/palo/internal/response"
	"github.com/GGP1/palo/pkg/auth"
	"github.com/GGP1/palo/pkg/auth/email"
	"github.com/GGP1/palo/pkg/model"
	"github.com/gorilla/mux"
)

// Login takes a user and authenticates it
func Login(s auth.Session, validatedList email.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.AlreadyLoggedIn(w, r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user := model.User{}

		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			response.Error(w, r, http.StatusUnauthorized, err)
			return
		}
		defer r.Body.Close()

		// Validate it has no empty values
		err = user.Validate("login")
		if err != nil {
			response.Error(w, r, http.StatusBadRequest, err)
			return
		}

		// Authenticate user
		err = s.Login(w, user.Email, user.Password)
		if err != nil {
			response.HTMLText(w, r, http.StatusUnauthorized, "error: Invalid email or password")
			return
		}

		// Check if the email is validated
		err = validatedList.Seek(user.Email)
		if err != nil {
			response.Error(w, r, http.StatusUnauthorized, errors.New("Please verify your email before logging in"))
			return
		}

		response.HTMLText(w, r, http.StatusOK, "You logged in!")
	}
}

// Logout removes the authentication cookie
func Logout(s auth.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie("SID")

		if c == nil {
			response.Error(w, r, http.StatusBadRequest, errors.New("You cannot log out without a session"))
			return
		}

		// Logout user from the session and delete cookies
		s.Logout(w, c)

		response.HTMLText(w, r, http.StatusOK, "You are now logged out.")
	}
}

// ValidateEmail is the email verification page
func ValidateEmail(pendingList, validatedList email.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validated bool
		token := mux.Vars(r)["token"]

		pList, err := pendingList.Read()
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		for k, v := range pList {
			if v == token {
				err := validatedList.Add(k, v)
				if err != nil {
					response.Error(w, r, http.StatusInternalServerError, err)
				}
				validated = true
			}
		}

		if !validated {
			response.Error(w, r, http.StatusInternalServerError, errors.New("An error ocurred when validating your email"))
			return
		}

		response.HTMLText(w, r, http.StatusOK, "You have successfully validated your email!")
	}
}
