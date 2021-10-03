package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nametaginc/tvm"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serveMain()
		return
	} else {
		if err := cliMain(); err != nil {
			fmt.Fprintln(os.Stderr, "ERROR", err.Error())
			os.Exit(1)
		}
	}
}

func serveMain() {
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	listenPort := flag.String("listen", "", "Run the server, listening on the specified port")
	rootURL := flag.String("url", "", "The URL of the server")
	oauth2ClientID := flag.String("oauth2-client-id", "", "")
	oauth2ClientSecret := flag.String("oauth2-client-secret", "", "")
	sessionMaxAgeSeconds := flag.Int("session-max-age", 120, "Number of seconds that an authentication session lasts")
	flag.Parse()

	if listenPort != nil && *listenPort != "" {
		rootURL, err := url.Parse(*rootURL)
		config := tvm.Config{
			RootURL:              *rootURL,
			OAuth2ClientID:      *oauth2ClientID,
			OAuth2ClientSecret:   *oauth2ClientSecret,
			SessionMaxAgeSeconds: *sessionMaxAgeSeconds,
		}
		srv, err := tvm.NewServer(config)
		if err != nil {
			log.Fatalf("cannot start server: %v", err)
		}
		srv.Store = tvm.LocalStore{Path: "data"}

		log.Printf("listening on %s", *listenPort)
		http.ListenAndServe(*listenPort, srv)
		return
	}

	// unknown command
	flag.Usage()
}

func cliMain() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	server := flag.String("s", "", "The URL of the TVM server")
	role := flag.String("r", "", "The role to use")

	store := tvm.FileClientStorage{
		Path: filepath.Join(os.Getenv("HOME"), ".config", "tvm", "tvm.json"),
	}
	state, err := store.Get(ctx)
	if err != nil {
		return err
	}
	if state == nil {
		state = &tvm.ClientState{}
	}

	var serverState tvm.ServerState
	{
		if *server == "" && len(state.Servers) == 1 {
			for s := range state.Servers {
				*server = s
			}
		}
		if *server == "" {
			return fmt.Errorf("Cannot infer server, specify -s")
		}
		if state.Servers != nil {
			serverState = state.Servers[*server]
		}
	}

	var credential tvm.Credential
	{
		if *role == "" && len(serverState.Roles) == 1 {
			for r := range serverState.Roles {
				*role = r
			}
		}
		if *role == "" {
			return fmt.Errorf("Cannot infer role, specify -r")
		}
		if serverState.Roles != nil {
			credential = serverState.Roles[*role]
		}
	}

	if !credential.Expires.IsZero() && time.Now().Before(credential.Expires) {
		fmt.Printf(
			"export TVM_AWS_ROLE=%s\n"+
			"export AWS_ACCESS_KEY_ID=%s\n"+
			"export AWS_SECRET_ACCESS_KEY=%s\n"+
			"export AWS_SESSION_TOKEN=%s\n",
			*role,
			credential.AccessKeyID,
			credential.SecretAccessKey,
			credential.SessionToken)
		return nil
	}

	doneCh := make(chan error)
	defer close(doneCh)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()


	type ServerResponse struct {
		AccountID string
		Role string
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
		Expires         time.Time
	}

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if errStr := r.URL.Query().Get("error"); errStr != "" {
			doneCh <- errors.New(errStr)
			return
		}
		*role = r.URL.Query().Get("role")
		credential.AccessKeyID = r.URL.Query().Get("access_key_id")
		credential.SecretAccessKey = r.URL.Query().Get("secret_access_key")
		credential.SessionToken = r.URL.Query().Get("session_token")
		credential.Expires, err  = time.Parse(time.RFC3339, r.URL.Query().Get("expiration"))
		if err != nil {
			doneCh <- err
			return
		}

		doneCh <- nil
	}))

	openURL, err := url.Parse(*server)
	if err != nil {
		return err
	}

	query := openURL.Query()
	query.Set("format", "cli")
	query.Set("port", listener.Addr().String())
	if *role != "" {
		query.Set("role", *role)
	}

	// TODO(ross): when I have internet access, find the library for this
	exec.Command("open", openURL.String()).Run()

	if err := <- doneCh; err != nil {
		return err
	}


	serverState.Roles[*role] = credential
	state.Servers[*server] = serverState

	if err := store.Put(ctx, *state); err != nil {
		return err
	}

	fmt.Printf(
		"export AWS_ROLE=%s\n"+
			"export AWS_ACCESS_KEY_ID=%s\n"+
			"export AWS_SECRET_ACCESS_KEY=%s\n"+
			"export AWS_SESSION_TOKEN=%s\n",
		*role,
		credential.AccessKeyID,
		credential.SecretAccessKey,
		credential.SessionToken)
	return nil
}
