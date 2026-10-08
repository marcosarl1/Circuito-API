package repository

import (
	"context"
	"testing"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/config"
	"github.com/marcosarl1/Circuito-API/internal/service"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// testStore sobe Mongo real em container. Sem Docker, pula em vez de falhar:
// a suíte unitária não pode exigir infraestrutura.
func testStore(t *testing.T) *Store {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		t.Skipf("sem docker para teste de repositório: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		_ = container.Terminate(stopCtx)
	})

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	store, err := Connect(ctx, config.Config{
		MongoURI:        uri,
		MongoDB:         "authtest",
		MongoCollection: "eventos",
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = store.Disconnect(context.Background()) })
	if err := store.EnsureAuthIndexes(ctx); err != nil {
		t.Fatalf("indexes: %v", err)
	}
	return store
}

// TestUserRoundTrip prova o ciclo completo contra BSON real. Foi um teste
// como este que faltou quando ObjectId quebrou o decode em string.
func TestUserRoundTrip(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	hash, err := service.HashPassword("senha-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	created, err := store.CreateUser(ctx, service.User{
		Username: "admincircuito", PasswordHash: hash, Role: service.RoleAdmin, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == bson.NilObjectID {
		t.Fatal("esperava ObjectID gerado")
	}

	byEmail, err := store.FindUserByUsername(ctx, "admincircuito")
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	byID, err := store.FindUserByID(ctx, byEmail.ID.Hex())
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if byID.Username != "admincircuito" || byID.Role != service.RoleAdmin {
		t.Fatalf("documento divergente: %+v", byID)
	}
	if err := service.VerifyPassword(byID.PasswordHash, "senha-123"); err != nil {
		t.Fatalf("credencial válida rejeitada após round-trip BSON: %v", err)
	}

	if _, err := store.CreateUser(ctx, service.User{Username: "admincircuito"}); err != ErrUserExists {
		t.Fatalf("duplicata deveria falhar com ErrUserExists, veio %v", err)
	}
}

// TestRefreshRotationRoundTrip prova consumo atômico e expiração no banco.
func TestRefreshRotationRoundTrip(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	_, hash, err := service.NewRefreshToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	now := time.Now()
	if err := store.StoreRefreshToken(ctx, service.RefreshToken{
		Hash: hash, UserID: "user-1", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}); err != nil {
		t.Fatalf("store: %v", err)
	}

	taken, err := store.TakeRefreshToken(ctx, hash)
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if taken.UserID != "user-1" {
		t.Fatalf("user divergente: %q", taken.UserID)
	}
	if _, err := store.TakeRefreshToken(ctx, hash); err != ErrInvalidRefreshToken {
		t.Fatalf("reuso deveria falhar, veio %v", err)
	}
}
