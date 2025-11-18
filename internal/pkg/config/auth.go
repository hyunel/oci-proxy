package config

import (
	"encoding/base64"
	"net/http"
)

type Auth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func (a *Auth) IsAuthenticated(r *http.Request) bool {
	if a.Username == "" || a.Password == "" {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	return user == a.Username && pass == a.Password
}

func (a *Auth) ApplyToRequest(req *http.Request) bool {
	if a.Username == "" || a.Password == "" {
		return false
	}
	auth := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	return true
}
