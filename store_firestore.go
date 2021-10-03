package tvm

import (
	"cloud.google.com/go/firestore"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Firestore struct {
	fs     *firestore.Client
}

var _ Store = Firestore{}  // Firestore must implement Store


func (s Firestore) GetSession(ctx context.Context, id string) (*Session, error) {
	dsnap, err := s.fs.Collection("sessions").Doc(id).Get(ctx)
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var rv Session
	if err := dsnap.DataTo(&rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (s Firestore) PutSession(ctx context.Context, session Session) (error) {
	_, err := s.fs.Collection("sessions").Doc(session.ID).Set(ctx, &session)
	return err
}

func (s Firestore) DeleteSession(ctx context.Context, id string) (error) {
	_, err := s.fs.Collection("sessions").Doc(id).Delete(ctx)
	return err
}

func (s Firestore) GetUser(ctx context.Context, id string) (*User, error) {
	dsnap, err := s.fs.Collection("users").Doc(id).Get(ctx)
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var rv User
	if err := dsnap.DataTo(&rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (s Firestore) PutUser(ctx context.Context, user User) error {
	_, err := s.fs.Collection("users").Doc(user.ID).Set(ctx, user)
	return err
}

func (s Firestore) DeleteUser(ctx context.Context, id string) (error) {
	_, err := s.fs.Collection("users").Doc(id).Delete(ctx)
	return err
}

func (s Firestore) ListUsers(ctx context.Context) ([]User, error) {
	docs, err := s.fs.Collection("users").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var users []User
	for _, dsnap := range docs {
		var user User
		if err := dsnap.DataTo(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
