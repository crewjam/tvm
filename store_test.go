package tvm

import (
	"context"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"os"
	"testing"
)

func TestLocalStore(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "")
	assert.Check(t, err)
	defer os.RemoveAll(tempdir)

	store := LocalStore{Path: tempdir}
	testStore(t, store)
}

func testStore(t *testing.T, store Store) {
	ctx := context.Background()

	t.Run("session", func(t *testing.T) {
		session, err := store.GetSession(ctx, "sessionid")
		assert.Error(t, err, "not found")
		assert.Check(t, is.Nil(session))

		err = store.PutSession(ctx, Session{ID: "sessionid", UserID: "userid"})
		assert.Check(t, err)

		session, err = store.GetSession(ctx, "sessionid")
		assert.Check(t, err)
		assert.Equal(t, "userid", session.UserID)

		err = store.DeleteSession(ctx, "sessionid")
		assert.Check(t, err)
		session, _ = store.GetSession(ctx, "sessionid")
		assert.Check(t, is.Nil(session))

		err = store.DeleteSession(ctx, "sessionid")
		assert.Error(t, err, "not found")
	})

	t.Run("user", func(t *testing.T) {
		user, err := store.GetUser(ctx, "userid")
		assert.Error(t, err, "not found")
		assert.Check(t, is.Nil(user))

		users, err := store.ListUsers(ctx)
		assert.Check(t, err)
		assert.Check(t, is.Len(users, 0))

		err = store.PutUser(ctx, User{ID: "userid", Admin: true})
		assert.Check(t, err)

		user, err = store.GetUser(ctx, "userid")
		assert.Check(t, err)
		assert.Equal(t, true, user.Admin)

		users, err = store.ListUsers(ctx)
		assert.Check(t, err)
		assert.Check(t, is.Len(users, 1))
		assert.Equal(t, "userid", users[0].ID)

		err = store.DeleteUser(ctx, "userid")
		assert.Check(t, err)
		user, _ = store.GetUser(ctx, "userid")
		assert.Check(t, is.Nil(user))

		users, err = store.ListUsers(ctx)
		assert.Check(t, err)
		assert.Check(t, is.Len(users, 0))

		err = store.DeleteUser(ctx, "userid")
		assert.Error(t, err, "not found")
	})
}
