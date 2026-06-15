package phoenix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSeedWords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.dat")
	// phoenixd writes the mnemonic as space-separated words on a single line.
	if err := os.WriteFile(path, []byte("abandon ability able about above absent\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := seedFilePath
	seedFilePath = path
	defer func() { seedFilePath = orig }()

	words, err := GetSeedWords()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 6 {
		t.Fatalf("expected 6 words, got %d: %v", len(words), words)
	}
	if words[0] != "abandon" || words[5] != "absent" {
		t.Fatalf("unexpected words: %v", words)
	}
}

func TestGetSeedWords_Missing(t *testing.T) {
	orig := seedFilePath
	seedFilePath = filepath.Join(t.TempDir(), "does-not-exist.dat")
	defer func() { seedFilePath = orig }()

	if _, err := GetSeedWords(); err == nil {
		t.Fatal("expected error for missing seed file")
	}
}

func TestGetSeedWords_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.dat")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := seedFilePath
	seedFilePath = path
	defer func() { seedFilePath = orig }()

	if _, err := GetSeedWords(); err == nil {
		t.Fatal("expected error for empty seed file")
	}
}
