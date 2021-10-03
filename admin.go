package tvm

import (
	_ "embed"
	"fmt"
	"github.com/aws/aws-sdk-go/service/iam"
	"html/template"
	"net/http"
)

// go:embed admin.tmpl.html
var adminTemplateStr string

var adminTemplate = template.Must(template.New("admin").Parse(adminTemplateStr))

func (s *Server) handleAdminRoot(w http.ResponseWriter, r *http.Request) {
	s.serveAdminRoot(w,r,"")
}

func (s *Server) handleAdminOp(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthorizedAdmin(r) {
		http.Redirect(w,r,"/?format=admin", http.StatusFound)
		return
	}

	user, err := s.Store.GetUser(r.Context(), r.FormValue("user"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var flash string

	switch r.FormValue("op") {
	case "add_role":
		user.Roles = append(user.Roles, r.FormValue("role"))
		flash = fmt.Sprintf("Added role %s to %s", r.FormValue("role"), user.ID)
	case "delete_role":
		roles := user.Roles[:0]
		for _, existingRole := range user.Roles {
			if existingRole == r.FormValue("role") {
				continue
			}
			roles = append(roles, existingRole)
		}
		user.Roles = roles
		flash = fmt.Sprintf("Removed role %s from %s", r.FormValue("role"), user.ID)
	case "delete_admin":
		user.Admin = false
		flash = fmt.Sprintf("Removed admin from %s", user.ID)

	case "add_admin":
		user.Admin = true
		flash = fmt.Sprintf("Added admin to %s", user.ID)

	case "reset_devices":
		user.U2FDevices = nil
		flash = fmt.Sprintf("Reset devices for %s", user.ID)

	default:
		http.Error(w, "unknown operation", http.StatusBadRequest)
		return
	}

	if err := s.Store.PutUser(r.Context(), *user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.serveAdminRoot(w, r, flash)
}

func (s *Server) serveAdminRoot(w http.ResponseWriter, r *http.Request, flash string) {
	if !s.isAuthorizedAdmin(r) {
		http.Redirect(w,r,"/?format=admin", http.StatusFound)
		return
	}

	users, err := s.Store.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var roles []string
	iamSvc := iam.New(nil)
	if err := iamSvc.ListRolesPagesWithContext(r.Context(), &iam.ListRolesInput{}, func(output *iam.ListRolesOutput, b bool) bool {
		for _, x := range output.Roles {
			roles = append(roles, *x.Arn)
		}
		return true
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	args := struct {
		Roles []string
		Users []User
		Flash string
	}{
		Roles: roles,
		Users: users,
		Flash: flash,
	}

	adminTemplate.Execute(w, args)
}

func (s *Server) isAuthorizedAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	session, err := s.Store.GetSession(r.Context(), cookie.Value)
	if err != nil {
		return false
	}
	user, err := s.Store.GetUser(r.Context(), session.UserID)
	if err != nil {
		return false
	}
	return user.Admin
}
