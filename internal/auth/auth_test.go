package auth

import "testing"

func TestHashPassword(t *testing.T) {
	pass := "12345"

	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	if hash == pass {
		t.Fatal("Hash should not equal the original password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	pass := "12345"
	wrongPass := "54321"

	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	match, err := CheckPasswordHash(pass, hash)
	if err != nil {
		t.Fatalf("Failed to check password: %v", err)
	}
	if !match {
		t.Fatal("Password should match")
	}

	// Test wrong password
	match, err = CheckPasswordHash(wrongPass, hash)
	if err != nil {
		t.Fatalf("Failed to check password: %v", err)
	}
	if match {
		t.Fatal("Wrong password should not match")
	}
}
func TestHashPasswordDifferentHashes(t *testing.T) {
	password := "samePassword"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("Two hashes of the same password should be different (different salts)")
	}

	match1, _ := CheckPasswordHash(password, hash1)
	match2, _ := CheckPasswordHash(password, hash2)

	if !match1 || !match2 {
		t.Fatal("Both hashes should match the original password")
	}
}
