package handler

import (
	"context"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
)

type fakeUserStore struct {
	users      map[string]*service.User
	byUsername map[string]*service.User
	tokens     map[string]*service.RefreshToken
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users:      map[string]*service.User{},
		byUsername: map[string]*service.User{},
		tokens:     map[string]*service.RefreshToken{},
	}
}

func (store *fakeUserStore) CountUsers(_ context.Context) (int64, error) {
	return int64(len(store.users)), nil
}

func (store *fakeUserStore) CreateUser(_ context.Context, user service.User) (*service.User, error) {
	if _, exists := store.byUsername[user.Username]; exists {
		return nil, repository.ErrUserExists
	}
	stored := user
	store.users[user.ID.Hex()] = &stored
	store.byUsername[user.Username] = &stored
	return &stored, nil
}

func (store *fakeUserStore) FindUserByUsername(_ context.Context, username string) (*service.User, error) {
	user, exists := store.byUsername[username]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (store *fakeUserStore) FindUserByID(_ context.Context, userID string) (*service.User, error) {
	user, exists := store.users[userID]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (store *fakeUserStore) StoreRefreshToken(_ context.Context, token service.RefreshToken) error {
	stored := token
	store.tokens[token.Hash] = &stored
	return nil
}

func (store *fakeUserStore) TakeRefreshToken(_ context.Context, hash string) (*service.RefreshToken, error) {
	token, exists := store.tokens[hash]
	if !exists {
		return nil, repository.ErrInvalidRefreshToken
	}
	delete(store.tokens, hash)
	if time.Now().After(token.ExpiresAt) {
		return nil, repository.ErrInvalidRefreshToken
	}
	return token, nil
}

func (store *fakeUserStore) RevokeRefreshToken(_ context.Context, hash string) error {
	delete(store.tokens, hash)
	return nil
}

func (store *fakeUserStore) UpdateUserPassword(_ context.Context, userID, hash string) error {
	user, exists := store.users[userID]
	if !exists {
		return repository.ErrUserNotFound
	}
	user.PasswordHash = hash
	return nil
}
