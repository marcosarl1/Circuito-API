package repository

import (
	"context"
	"errors"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	// ErrUserExists is returned when the username is already registered.
	ErrUserExists = errors.New("user already exists")
	// ErrUserNotFound covers unknown users without distinguishing causes.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidRefreshToken covers unknown, expired and already-used tokens
	// with a single value, so callers cannot distinguish them (no oracle).
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

func objectIDFromHex(raw string) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(raw)
}

func (store *Store) users() *mongo.Collection {
	return store.DB.Collection("users")
}

func (store *Store) refreshTokens() *mongo.Collection {
	return store.DB.Collection("refresh_tokens")
}

// EnsureAuthIndexes creates the email uniqueness and the refresh-token TTL
// (expired sessions vanish without a cleanup job).
func (store *Store) EnsureAuthIndexes(requestContext context.Context) error {
	requestContext, cancel := context.WithTimeout(requestContext, 30*time.Second)
	defer cancel()

	if _, err := store.users().Indexes().CreateOne(requestContext, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	_, err := store.refreshTokens().Indexes().CreateOne(requestContext, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

// CountUsers reports how many accounts exist (bootstrap check).
func (store *Store) CountUsers(requestContext context.Context) (int64, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	return store.users().CountDocuments(requestContext, bson.M{})
}

// CreateUser inserts a user with an already-hashed password.
func (store *Store) CreateUser(requestContext context.Context, user service.User) (*service.User, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	if _, err := store.users().InsertOne(requestContext, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByUsername loads by the unique username (login path).
func (store *Store) FindUserByUsername(requestContext context.Context, username string) (*service.User, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	var user service.User
	if err := store.users().FindOne(requestContext, bson.M{"username": username}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByID loads by id (per-request authentication path).
func (store *Store) FindUserByID(requestContext context.Context, userID string) (*service.User, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	objectID, err := objectIDFromHex(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	var user service.User
	if err := store.users().FindOne(requestContext, bson.M{"_id": objectID}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// StoreRefreshToken persists the token hash with its expiry.
func (store *Store) StoreRefreshToken(requestContext context.Context, token service.RefreshToken) error {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	_, err := store.refreshTokens().InsertOne(requestContext, token)
	return err
}

// TakeRefreshToken atomically consumes a token (rotation): it is returned
// once and deleted, so replay of an old token always fails closed.
func (store *Store) TakeRefreshToken(requestContext context.Context, hash string) (*service.RefreshToken, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	var token service.RefreshToken
	err := store.refreshTokens().FindOneAndDelete(requestContext, bson.M{"_id": hash}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}
	return &token, nil
}

// RevokeRefreshToken deletes one token idempotently (logout).
func (store *Store) RevokeRefreshToken(requestContext context.Context, hash string) error {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	_, err := store.refreshTokens().DeleteOne(requestContext, bson.M{"_id": hash})
	return err
}

// UpdateUserPassword replaces the (already hashed) password.
func (store *Store) UpdateUserPassword(requestContext context.Context, userID, hash string) error {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	objectID, err := objectIDFromHex(userID)
	if err != nil {
		return ErrUserNotFound
	}
	result, err := store.users().UpdateOne(
		requestContext,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"password_hash": hash}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrUserNotFound
	}
	return nil
}
