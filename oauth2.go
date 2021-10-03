package tvm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

func (s *Server) handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		fmt.Fprintln(w, "bad session cookie")
		return
	}
	session, err := s.Store.GetSession(r.Context(), cookie.Value)
	if err != nil {
		fmt.Fprintln(w, "bad session")
		return
	}
	if r.URL.Query().Get("state") != session.OAuth2State {
		fmt.Fprintln(w, "bad state")
		return
	}

	token, err := s.OAuth2.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		fmt.Fprintln(w, "bad code")
		return
	}

	idTokenStr, _ := token.Extra("id_token").(string)
	if idTokenStr == "" {
		panic("token does not contain 'id_token'")
	}

	var idToken IDToken
	if err := ParseJWTWithoutVerifying(idTokenStr, &idToken); err != nil {
		panic(err)
	}

	session.OAuth2State = ""
	session.UserID = idToken.Email
	if err := s.Store.PutSession(r.Context(), *session); err != nil {
		panic(fmt.Errorf("cannot store session: %s", err))
	}

	user, err := s.Store.GetUser(r.Context(), session.UserID)
	if err != nil {
		fmt.Fprintln(w, "cannot fetch user: %w", err)
		return
	}
	if user == nil {
		user = &User{
			ID: session.UserID,
		}
		if err := s.Store.PutUser(r.Context(), *user); err != nil {
			fmt.Fprintln(w, "cannot create user: %w", err)
			return
		}
	}

	if len(user.U2FDevices) == 0 {
		http.Redirect(w,r,"/u2f/register", http.StatusFound)
		return
	}

	s.sendU2FChallenge(w,r,*session, *user)
}


// IDToken represents an OIDC ID token returned in the `id_token` field from an OAuth 2.0
// token endpoint.
type IDToken struct {
	Email             string `json:"email"`
	Aud               string `json:"aud"`
	Iss               string `json:"iss"`
	Iat               int    `json:"iat"`
	Nbf               int    `json:"nbf"`
	Exp               int    `json:"exp"`
	Aio               string `json:"aio"`
	Name              string `json:"name"`
	Nonce             string `json:"nonce"`
	Oid               string `json:"oid"`
	PreferredUsername string `json:"preferred_username"`
	Sub               string `json:"sub"`
	Tid               string `json:"tid"`
	Uti               string `json:"uti"`
	UPN               string `json:"upn"` // not sure if this is ever present
	Ver               string `json:"ver"`
	Hd                string `json:"hd"` // google domain
}

// ParseJWTWithoutVerifying fills in out with the unmarshalled payload of jwt.
//
// Yes, the name is clumsy but I don't want you to be able to forget that the
// signature is not validated.
func ParseJWTWithoutVerifying(jwt string, out interface{}) error {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return fmt.Errorf("expected JWT")
	}

	payloadBuf, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return errors.Wrap(err, "cannot decode JWT payload")
	}

	if err := json.Unmarshal(payloadBuf, out); err != nil {
		return errors.Wrap(err, "cannot parse JWT payload")
	}

	return nil
}
