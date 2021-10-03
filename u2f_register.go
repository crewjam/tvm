package tvm

import (
	"encoding/json"
	"fmt"
	"github.com/tstranex/u2f"
	"log"
	"net/http"
)

func (s *Server) handleU2FRegister(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("resp") != "" {
		s.handleU2FRegisterSigned(w, r)
		return
	}

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

	user, err := s.Store.GetUser(r.Context(), session.UserID)
	if err != nil {
		fmt.Fprintln(w, "bad user")
		return
	}

	if len(user.U2FDevices) > 0 && !session.U2F {
		fmt.Fprintln(w, "need u2f")
		return
	}

	c, err := u2f.NewChallenge(s.u2fAppID(), []string{s.u2fAppID()})
	if err != nil {
		log.Printf("u2f.NewChallenge error: %v", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	session.U2FChallenge = c

	req := u2f.NewWebRegisterRequest(c, user.U2FRegistrations())
	reqJSON, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w,
		`
<!DOCTYPE html>
<html>
  <head>
    <!-- The original u2f-api.js code can be found here:
    https://github.com/google/u2f-ref-code/blob/master/u2f-gae-demo/war/js/u2f-api.js -->
    <script type="text/javascript" src="/u2f-api.js"></script>
  </head>
<body>
<h1>Press the button on your Yubikey</h1>
<script>
function didRegister(resp) {
	console.log('didRegister', resp)
	window.location.assign("/u2f/register?resp=" + encodeURIComponent(JSON.stringify(resp)))
}
function register() {
  console.log('register()')
  var req = %s;
  u2f.register(req.appId, req.registerRequests, req.registeredKeys || [], didRegister, 30);
}
document.addEventListener('DOMContentLoaded', register)
</script>
</body>
</html>
`, reqJSON)

}


func (s *Server) handleU2FRegisterSigned(w http.ResponseWriter, r *http.Request) {
	var regResp u2f.RegisterResponse
	if err := json.Unmarshal([]byte(r.URL.Query().Get("resp")), &regResp); err != nil {
		panic(err)
	}

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

	user, err := s.Store.GetUser(r.Context(), session.UserID)
	if err != nil {
		fmt.Fprintln(w, "bad user")
		return
	}

	config := &u2f.Config{
		// Chrome 66+ doesn't return the device's attestation
		// certificate by default.
		SkipAttestationVerify: true,
	}

	reg, err := u2f.Register(regResp, *session.U2FChallenge, config)
	if err != nil {
		log.Printf("u2f.Register error: %v", err)
		http.Error(w, "error verifying response", http.StatusInternalServerError)
		return
	}

	user.U2FDevices = append(user.U2FDevices, U2FDevice{
		Registration: *reg,
		Counter:      0,
	})
	if err := s.Store.PutUser(r.Context(), *user); err != nil {
		panic(err)
	}

	session.U2FChallenge = nil
	if err := s.Store.PutSession(r.Context(), *session); err != nil {
		panic(err)
	}

	fmt.Fprintln(w, "OK")
}