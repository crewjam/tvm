package tvm

import (
	_ "embed"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	"goji.io"
	"goji.io/pat"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	RootURL              url.URL
	OAuth2ClientID       string
	OAuth2ClientSecret   string
	SessionMaxAgeSeconds int
	CredentialLifetimeSeconds int
}

func NewServer(config Config) (*Server, error) {
	s := Server{Mux: goji.NewMux(), Config: config}

	redirectURL := config.RootURL
	redirectURL.Path = "/oauth2/callback"
	s.OAuth2 = oauth2.Config{
		ClientID:     config.OAuth2ClientID,
		ClientSecret: config.OAuth2ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL.String(),
		Scopes:       []string{"openid", "email"},
	}

	s.Mux.HandleFunc(pat.Get("/"), s.handleGetToken)

	s.Mux.HandleFunc(pat.Get("/oauth2/callback"), s.handleOAuth2Callback)
	s.Mux.HandleFunc(pat.Get("/u2f/sign"), s.handleU2FSigned)
	s.Mux.HandleFunc(pat.Get("/u2f/register"), s.handleU2FRegister)

	s.Mux.HandleFunc(pat.Get("/admin"), s.handleAdminRoot)
	s.Mux.HandleFunc(pat.Post("/admin"), s.handleAdminOp)

	s.Mux.HandleFunc(pat.Get("/u2f-api.js"), handleU2FApiJS)

	return &s, nil
}

//go:embed u2f-api.js
var u2fAPIJS []byte

func handleU2FApiJS(w http.ResponseWriter, r *http.Request) {
	w.Write(u2fAPIJS)
}

type Server struct {
	*goji.Mux
	OAuth2 oauth2.Config
	Store  Store
	Config Config
}

func (s *Server) handleGetToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		s.newSession(w, r)
		return
	}

	session, err := s.Store.GetSession(r.Context(), cookie.Value)
	if err != nil {
		s.newSession(w, r)
		return
	}

	if session.UserID == "" {
		s.Store.DeleteSession(r.Context(), session.ID)
		s.newSession(w, r)
		return
	}

	if !session.U2F {
		s.Store.DeleteSession(r.Context(), session.ID)
		s.newSession(w, r)
		return
	}

	user, err := s.Store.GetUser(r.Context(), session.UserID)
	if err != nil {
		s.Store.DeleteSession(r.Context(), session.ID)
		s.newSession(w, r)
		return
	}

	if r.URL.Query().Get("format") == "admin" {
		if !user.Admin {
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			http.Redirect(w, r, "/admin", http.StatusFound)
		}
		return
	}

	desiredRole := r.URL.Query().Get("role")
	if desiredRole == "" && len(user.Roles) == 1 {
		desiredRole = user.Roles[0]
	}

	roleIsOK := false
	for _, role := range user.Roles {
		if role == desiredRole {
			roleIsOK = true
		}
	}
	if !roleIsOK {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "role %q is not allowed\n", desiredRole)
		return
	}

	stsSvc := sts.New(nil)
	assumeRoleOutput, err := stsSvc.AssumeRoleWithContext(r.Context(), &sts.AssumeRoleInput{
		DurationSeconds:   aws.Int64(int64(s.Config.CredentialLifetimeSeconds)),
		RoleArn:           &desiredRole,
		RoleSessionName:   aws.String(fmt.Sprintf("tvm:%s", user.ID)),
	})
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, "sts.AssumeRole failed")
		return
	}

	if r.URL.Query().Get("format") == "cli" {
		query := url.Values{
			"role":              {desiredRole },
			"access_key_id":     {*assumeRoleOutput.Credentials.AccessKeyId},
			"secret_access_key": {*assumeRoleOutput.Credentials.SecretAccessKey},
			"session_token":     {*assumeRoleOutput.Credentials.SessionToken},
			"expiration":        {assumeRoleOutput.Credentials.Expiration.Format(time.RFC3339)},
		}

		port, err := strconv.Atoi(r.URL.Query().Get("port"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "cannot parse port")
			return
		}
		nextURL := fmt.Sprintf("http://localhost:%d/?%s", port, query.Encode())
		http.Redirect(w,r,nextURL, http.StatusFound)
		return
	}

	if r.URL.Query().Get("format") == "sh" {
		fmt.Fprintf(w, "export AWS_ACCESS_KEY_ID=%s\n"+
			"export AWS_SECRET_ACCESS_KEY=%s\n"+
			"export AWS_SESSION_TOKEN=%s\n", *assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken)
		return
	}


	// TODO(ross): construct an AWS console URL from the credentials
}
