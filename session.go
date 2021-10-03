package tvm

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
)

func (s *Server) newSession(w http.ResponseWriter, r *http.Request)  {
	id := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		panic(err)
	}

	oauth2state := make([]byte, 32)
	_, err = io.ReadFull(rand.Reader, oauth2state)
	if err != nil {
		panic(err)
	}

	session := Session{
		ID: base64.RawURLEncoding.EncodeToString(id),
		OAuth2State: base64.RawURLEncoding.EncodeToString(oauth2state),
		Params: r.URL.Query(),
	}
	if err := s.Store.PutSession(r.Context(), session); err != nil {
		panic(err)
	}

	cookie := http.Cookie{
		Name:     "session",
		Value:    session.ID,
		MaxAge:   s.Config.SessionMaxAgeSeconds,
		//HttpOnly: true,
		//Secure:   false,
		//SameSite: http.SameSiteStrictMode,  // TODO(ross): confirm this
		//Path:     "/",
	}
	http.SetCookie(w, &cookie)

	redirectURL := s.OAuth2.AuthCodeURL(session.OAuth2State)
	http.Redirect(w,r, redirectURL, http.StatusFound)
}

