package tvm

import (
	"context"
	"errors"
	"github.com/tstranex/u2f"
	"net/url"
)

type Session struct {
	ID string
	UserID string
	Params url.Values
	U2F bool
	OAuth2State string
	U2FChallenge *u2f.Challenge
}

type User struct {
	ID string
	Roles []string
	U2FDevices []U2FDevice
	Admin bool
}

func (u User) U2FRegistrations() []u2f.Registration {
	var rv []u2f.Registration
	for _, dev := range u.U2FDevices {
		rv = append(rv, dev.Registration)
	}
	return rv
}

type U2FDevice struct {
	Registration u2f.Registration
	Counter uint32
}

type Store interface {
	GetSession(ctx context.Context, id string) (*Session, error)
	PutSession(ctx context.Context, session Session) (error)
	DeleteSession(ctx context.Context, id string) (error)
	GetUser(ctx context.Context, id string) (*User, error)
	PutUser(ctx context.Context, user User) error
	DeleteUser(ctx context.Context, id string) (error)
	ListUsers(ctx context.Context) ([]User, error)
}

var ErrNotFound = errors.New("not found")
