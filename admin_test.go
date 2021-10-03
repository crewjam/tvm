package tvm

import (
	"bytes"
	"context"
	"gotest.tools/golden"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestAdmin(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "")
	assert.Check(t, err)
	defer os.RemoveAll(tempdir)

	ctx := context.Background()

	s, err := NewServer(Config{})
	assert.Check(t, err)
	s.Store = LocalStore{Path: tempdir}

	err = s.Store.PutUser(ctx, User{ID: "userid", Admin: true})
	assert.Check(t, err)
	err = s.Store.PutSession(ctx, Session{ID: "sessionid", UserID: "userid"})
	assert.Check(t, err)

	t.Run("authenticated", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/admin", nil)
		r.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "sessionid",
		})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 200, w.Code)
		golden.Assert(t, string(buf.Bytes()), "authenticated")
	})

	t.Run("requires auth", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/admin", nil)
		r.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "badsessionid",
		})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/?format=admin", w.Header().Get("Location"))
	})

	t.Run("requires admin", func(t *testing.T) {
		err = s.Store.PutUser(ctx, User{ID: "userid", Admin: false})
		assert.Check(t, err)

		r := httptest.NewRequest("GET", "/admin", nil)
		r.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "sessionid",
		})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/?format=admin", w.Header().Get("Location"))
	})
}


func TestAdminOp(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "")
	assert.Check(t, err)
	defer os.RemoveAll(tempdir)

	ctx := context.Background()

	s, err := NewServer(Config{})
	assert.Check(t, err)
	s.Store = LocalStore{Path: tempdir}

	err = s.Store.PutUser(ctx, User{ID: "userid", U2FDevices: []U2FDevice{{Counter: 42}}})
	assert.Check(t, err)
	err = s.Store.PutUser(ctx, User{ID: "adminuser", Admin: true})
	assert.Check(t, err)
	err = s.Store.PutSession(ctx, Session{ID: "sessionid", UserID: "adminuser"})
	assert.Check(t, err)

	t.Run("requires auth", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", nil)
		r.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "badsessionid",
		})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/?format=admin", w.Header().Get("Location"))
	})

	t.Run("requires admin", func(t *testing.T) {
		err = s.Store.PutSession(ctx, Session{ID: "nonadminsessionid", UserID: "userid"})
		assert.Check(t, err)

		r := httptest.NewRequest("POST", "/admin/op", nil)
		r.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "nonadminsessionid",
		})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/?format=admin", w.Header().Get("Location"))
	})


	t.Run("add role", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"add_role"},
				"user": {"userid"},
				"role": {"myrole"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 200, w.Code)

		newUser, err := s.Store.GetUser(ctx, "userid")
		assert.Check(t, err)
		assert.Equal(t, newUser.Roles, []string{"myrole"})
	})


	t.Run("delete_role", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"delete_role"},
				"user": {"userid"},
				"role": {"myrole"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 200, w.Code)

		newUser, err := s.Store.GetUser(ctx, "userid")
		assert.Check(t, err)
		assert.Equal(t, newUser.Roles, []string{})
	})

	t.Run("add_admin", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"add_admin"},
				"user": {"userid"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 200, w.Code)

		newUser, err := s.Store.GetUser(ctx, "userid")
		assert.Check(t, err)
		assert.Equal(t, newUser.Admin, true)
	})
	t.Run("delete_admin", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"delete_admin"},
				"user": {"userid"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 200, w.Code)

		newUser, err := s.Store.GetUser(ctx, "userid")
		assert.Check(t, err)
		assert.Equal(t, newUser.Admin, false)
	})

	t.Run("reset_devices", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"reset_devices"},
				"user": {"userid"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 200, w.Code)

		newUser, err := s.Store.GetUser(ctx, "userid")
		assert.Check(t, err)
		assert.Check(t, is.Len(newUser.U2FDevices, 0))
	})

	t.Run("unknown operation", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"unknown_operation"},
				"user": {"userid"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 400, w.Code)
	})
	t.Run("unknown user", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/admin/op", strings.NewReader(
			url.Values{
				"op": {"delete_admin"},
				"user": {"baduserid"},
			}.Encode()))
		r.AddCookie(&http.Cookie{Name:  "session", Value: "sessionid"})

		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		buf := bytes.NewBuffer(nil)
		w.Result().Write(buf)
		assert.Equal(t, 400, w.Code)
	})
}