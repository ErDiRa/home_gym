package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/modules/identity/app"
)

func TestTokens(t *testing.T) {
	tokens, err := NewTokens("test-secret-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	other, _ := NewTokens("another-secret-another")
	login := time.Now().Add(-time.Hour).Truncate(time.Second)
	claims := app.Claims{UserID: uuid.New(), AuthTime: login, Remember: true}

	valid, _ := tokens.Issue(claims, time.Now().Add(time.Hour))
	got, err := tokens.Parse(valid)
	if err != nil || got.UserID != claims.UserID || !got.AuthTime.Equal(login) || !got.Remember {
		t.Errorf("Parse = %+v, %v; want %+v", got, err, claims)
	}

	expired, _ := tokens.Issue(claims, time.Now().Add(-time.Hour))
	foreign, _ := other.Issue(claims, time.Now().Add(time.Hour))
	for name, token := range map[string]string{"expired": expired, "wrong signature": foreign, "garbage": "nope"} {
		if _, err := tokens.Parse(token); err == nil {
			t.Errorf("Parse of a %s token succeeded", name)
		}
	}

	if _, err := NewTokens("short"); err == nil {
		t.Error("NewTokens accepted a short secret")
	}
}
