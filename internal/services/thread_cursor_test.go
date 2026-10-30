package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestThreadCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 7, 11, 3, 4, 5, 123456789, time.UTC)
	id := uuid.MustParse("018f6a10-1111-7000-8000-000000000001")

	decodedAt, decodedID, err := decodeThreadCursor(encodeThreadCursor(createdAt, id))
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !decodedAt.Equal(createdAt) {
		t.Fatalf("created_at = %s, want %s", decodedAt, createdAt)
	}
	if uuid.UUID(decodedID.Bytes) != id {
		t.Fatalf("id = %s, want %s", uuid.UUID(decodedID.Bytes), id)
	}
}

func TestThreadCursorRejectsGarbage(t *testing.T) {
	if _, _, err := decodeThreadCursor("not-a-cursor"); err == nil {
		t.Fatal("expected invalid cursor")
	}
}

func TestNormalizeThreadContent(t *testing.T) {
	got, err := normalizeThreadContent("  hello  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("content = %q", got)
	}
	if _, err := normalizeThreadContent("   "); err == nil {
		t.Fatal("expected blank content to fail")
	}
}
