package tvm

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

type ClientStorage interface {
	Get(ctx context.Context) (*ClientState, error)
	Put(ctx context.Context, state ClientState) error
}

type ClientState struct {
	// keys are server URLs
	Servers map[string]ServerState
}

type ServerState struct {
	// keys are roles
	Roles map[string]Credential
}

type Credential struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expires         time.Time
}

var _ ClientStorage = FileClientStorage{}

type FileClientStorage struct {
	Path string
}

func (f FileClientStorage) Get(ctx context.Context) (*ClientState, error) {
	buf, err := ioutil.ReadFile(f.Path)
	if os.IsNotExist(err) {
		return &ClientState{}, nil
	}
	if err != nil {
		return nil, err
	}
	var rv ClientState
	if err := json.Unmarshal(buf, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (f FileClientStorage) Put(ctx context.Context, data ClientState)  (error) {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f.Path, buf, 0600)
}

