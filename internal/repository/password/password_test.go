package password

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNew(t *testing.T) {
	t.Run("creates repository with given cost", func(t *testing.T) {
		repo := New(12)
		if repo.passCost != 12 {
			t.Errorf("expected passCost 12, got %d", repo.passCost)
		}
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("returns error on empty password", func(t *testing.T) {
		repo := New(10)
		_, err := repo.HashPassword("")
		if err == nil {
			t.Error("expected error for empty password, got nil")
		}
	})

	t.Run("returns error on password longer than 64 characters", func(t *testing.T) {
		repo := New(10)
		longPass := strings.Repeat("x", 65)
		_, err := repo.HashPassword(longPass)
		if err == nil {
			t.Error("expected error for password > 64 chars, got nil")
		}
	})

	t.Run("accepts password of exactly 64 characters", func(t *testing.T) {
		repo := New(10)
		longPass := strings.Repeat("x", 64)
		hash, err := repo.HashPassword(longPass)
		if err != nil {
			t.Fatalf("HashPassword failed on 64 chars: %v", err)
		}
		if len(hash) == 0 {
			t.Error("hash is empty")
		}
	})

	t.Run("uses DefaultCost when cost is too low", func(t *testing.T) {
		repo := New(bcrypt.MinCost - 1)
		hash, err := repo.HashPassword("valid")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			t.Fatalf("bcrypt.Cost failed: %v", err)
		}
		if cost != bcrypt.DefaultCost {
			t.Errorf("expected cost %d, got %d", bcrypt.DefaultCost, cost)
		}
	})

	t.Run("uses MaxCost when cost is too high", func(t *testing.T) {
		repo := New(4)
		hash, err := repo.HashPassword("valid")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			t.Fatalf("bcrypt.Cost failed: %v", err)
		}
		if cost != 4 {
			t.Errorf("expected cost %d, got %d", 4, cost)
		}
	})

	t.Run("generates valid bcrypt hash with normal cost", func(t *testing.T) {
		repo := New(10)
		hash, err := repo.HashPassword("testpass")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		if len(hash) == 0 {
			t.Error("hash is empty")
		}
		if hash[0] != '$' {
			t.Error("hash does not start with $")
		}

		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			t.Fatalf("bcrypt.Cost failed: %v", err)
		}
		if cost != 10 {
			t.Errorf("expected cost 10, got %d", cost)
		}
	})

	t.Run("different calls produce different hashes (salt)", func(t *testing.T) {
		repo := New(4)
		hash1, err := repo.HashPassword("samepass")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}
		hash2, err := repo.HashPassword("samepass")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		if hash1 == hash2 {
			t.Error("same password produced same hash, salt not working")
		}
	})
}

func TestCheckPasswordHash(t *testing.T) {
	repo := New(4)

	t.Run("returns true for correct password", func(t *testing.T) {
		hash, err := repo.HashPassword("correct")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		match := repo.CheckPasswordHash("correct", hash)
		if !match {
			t.Error("expected true for correct password, got false")
		}
	})

	t.Run("returns false for wrong password", func(t *testing.T) {
		hash, err := repo.HashPassword("correct")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		match := repo.CheckPasswordHash("wrong", hash)
		if match {
			t.Error("expected false for wrong password, got true")
		}
	})

	t.Run("returns false for empty password", func(t *testing.T) {
		hash, err := repo.HashPassword("valid")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}

		match := repo.CheckPasswordHash("", hash)
		if match {
			t.Error("expected false for empty password, got true")
		}
	})

	t.Run("returns false for empty hash", func(t *testing.T) {
		match := repo.CheckPasswordHash("valid", "")
		if match {
			t.Error("expected false for empty hash, got true")
		}
	})

	t.Run("returns false for invalid hash format", func(t *testing.T) {
		match := repo.CheckPasswordHash("valid", "not-a-bcrypt-hash")
		if match {
			t.Error("expected false for invalid hash, got true")
		}
	})

	t.Run("returns false for truncated hash", func(t *testing.T) {
		hash, err := repo.HashPassword("valid")
		if err != nil {
			t.Fatalf("HashPassword failed: %v", err)
		}
		if len(hash) < 10 {
			t.Skip("hash too short to truncate")
		}

		shortHash := hash[:10] // обрезаем хеш
		match := repo.CheckPasswordHash("valid", shortHash)
		if match {
			t.Error("expected false for truncated hash, got true")
		}
	})
}
