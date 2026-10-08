package service

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestLoginRequestValidation(t *testing.T) {
	valid := LoginRequest{Username: "admincircuito", Password: "segredo-123"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid rejected: %v", err)
	}

	invalid := []LoginRequest{
		{Username: "", Password: "segredo-123"},
		{Username: "ab", Password: "segredo-123"},
		{Username: "com espaço", Password: "segredo-123"},
		{Username: "admincircuito", Password: ""},
		{Username: "  ", Password: "x"},
	}
	for index, request := range invalid {
		if err := request.Validate(); err == nil {
			t.Fatalf("case %d accepted: %+v", index, request)
		}
	}
}

func TestRefreshRequestValidation(t *testing.T) {
	raw, _, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if err := (RefreshRequest{RefreshToken: raw}).Validate(); err != nil {
		t.Fatalf("valid rejected: %v", err)
	}

	for _, value := range []string{"", "curto", strings.Repeat("z", 64), strings.Repeat("0", 63)} {
		if err := (RefreshRequest{RefreshToken: value}).Validate(); err == nil {
			t.Fatalf("accepted garbage: %q", value)
		}
	}
}

func TestPasswordPolicy(t *testing.T) {
	if err := ValidateNewPassword("1234567"); err == nil {
		t.Fatal("7 chars accepted")
	}
	if err := ValidateNewPassword("12345678"); err != nil {
		t.Fatalf("8 chars rejected: %v", err)
	}
}

func TestPublicUserHidesHash(t *testing.T) {
	oid := bson.NewObjectID()
	user := User{ID: oid, Username: "admin", PasswordHash: "hash-secreto", CreatedAt: time.Now()}
	public := user.Public()
	if public.ID != oid.Hex() || public.Username != "admin" {
		t.Fatalf("projection lost fields: %+v", public)
	}
}
