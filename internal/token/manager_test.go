package token

import (
	"errors"
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	manager := New("12345678901234567890123456789012")
	value, err := manager.Issue("mini", 42, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Parse(value, "mini")
	if err != nil {
		t.Fatal(err)
	}
	if claims.SubjectID != 42 {
		t.Fatalf("expected subject 42, got %d", claims.SubjectID)
	}
	if _, err := manager.Parse(value, "admin"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}
