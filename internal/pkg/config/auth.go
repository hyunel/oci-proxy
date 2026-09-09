package config

import (
	"crypto/subtle"
	"net/http"
)

type Auth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func (a *Auth) IsSet() bool {
	return a != nil && (a.Username != "" || a.Password != "")
}

func (a *Auth) IsAuthenticated(r *http.Request) bool {
	if !a.IsSet() {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(user), []byte(a.Username))&
		subtle.ConstantTimeCompare([]byte(pass), []byte(a.Password)) == 1
}

func (a *Auth) ApplyToRequest(req *http.Request) bool {
	if !a.IsSet() {
		return false
	}
	req.SetBasicAuth(a.Username, a.Password)
	return true
}
