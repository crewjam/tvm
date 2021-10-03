package tvm

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type LocalStore struct {
	Path string
}

var _ Store = LocalStore{}  // LocalStore must implement Store

func (s LocalStore) GetSession(ctx context.Context, id string) (*Session, error) {
	path := filepath.Join(s.Path, "sessions", id+".json")
	buf, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	log.Printf("GetSession: %s", string(buf))

	var rv Session
	if err := json.Unmarshal(buf, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (s LocalStore) PutSession(ctx context.Context, session Session) (error) {
	path := filepath.Join(s.Path, "sessions", session.ID+".json")
	buf, err := json.Marshal(session)
	if err != nil {
		return err
	}
	log.Printf("PutSession: %s", string(buf))

	os.MkdirAll(filepath.Dir(path), 0700)
	return ioutil.WriteFile(path, buf, 0600)
}

func (s LocalStore) DeleteSession(ctx context.Context, id string) (error) {
	path := filepath.Join(s.Path, "sessions", id+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}

func (s LocalStore) GetUser(ctx context.Context, id string) (*User, error) {
	path := filepath.Join(s.Path, "users", id+".json")
	buf, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	var rv User
	if err := json.Unmarshal(buf, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (s LocalStore) PutUser(ctx context.Context, user User) error {
	path := filepath.Join(s.Path, "users", user.ID+".json")
	buf, err := json.Marshal(user)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(path), 0700)
	return ioutil.WriteFile(path, buf, 0600)
}

func (s LocalStore) DeleteUser(ctx context.Context, id string) error {
	path := filepath.Join(s.Path, "users", id+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}

func (s LocalStore) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	files, err := os.ReadDir(filepath.Join(s.Path, "users"))
	if err != nil {
		if os.IsNotExist(err) {
			return users, nil
		}
		return nil, err
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			user, err := s.GetUser(ctx, strings.TrimSuffix(file.Name(), ".json"))
			if err != nil {
				return nil, err
			}
			users = append(users, *user)
		}
	}
	return users, nil
}
