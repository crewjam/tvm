package tvm

import (
	"encoding/json"
	"fmt"
	"github.com/tstranex/u2f"
	"log"
	"net/http"
)

func (s *Server) u2fAppID() string {
	u := s.Config.RootURL
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}

func (s *Server) sendU2FChallenge(w http.ResponseWriter, r *http.Request, session Session, user User) {
	// Send authentication request to the browser.
	c, err := u2f.NewChallenge(s.u2fAppID(), []string{s.u2fAppID()})
	if err != nil {
		panic(err)
	}

	session.U2FChallenge = c
	if err := s.Store.PutSession(r.Context(), session); err != nil {
		panic(err)
	}

	req := c.SignRequest(user.U2FRegistrations())
	if err != nil {
		panic(err)
	}
	reqJSON, _ := json.Marshal(*req)

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
function didSign(resp) {
	window.location.assign("/u2f/signed?resp=" + encodeURIComponent(JSON.stringify(resp)))
}
function sign() {
  var req = %s;
  u2f.sign(req.appId, req.challenge, req.registeredKeys, didSign, 30);
}
document.onload = sign
</script>
</body>
</html>
`, reqJSON)

}

func (s *Server) handleU2FSigned(w http.ResponseWriter, r *http.Request) {

	var signResp u2f.SignResponse
	if err := json.Unmarshal([]byte(r.URL.Query().Get("resp")), &signResp); err != nil {
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

	if session.U2FChallenge == nil {
		http.Error(w, "challenge missing", http.StatusBadRequest)
		return
	}

	user, err := s.Store.GetUser(r.Context(), session.UserID)
	if err != nil {
		fmt.Fprintln(w, "bad session")
		return
	}

	for i, device := range user.U2FDevices {
		newCounter, authErr := device.Registration.Authenticate(signResp, *session.U2FChallenge, device.Counter)
		if authErr != nil {
			log.Printf("u2f: authenticate: %s", authErr)
			continue
		}

		log.Printf("newCounter: %d", newCounter)
		device.Counter = newCounter

		user.U2FDevices[i].Counter = newCounter
		if err := s.Store.PutUser(r.Context(), *user); err != nil {
			panic(err)
		}

		session.U2FChallenge = nil
		session.U2F = true
		if err := s.Store.PutSession(r.Context(), *session); err != nil {
			panic(err)
		}

		break
	}

	if !session.U2F {
		fmt.Fprintln(w, "u2f sign failed")
		return
	}

	http.Redirect(w, r, "/?" + session.Params.Encode(), http.StatusFound)
}
