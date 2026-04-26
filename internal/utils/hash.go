package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// SHA256File computes the SHA-256 hex digest of the file at path.
// It streams the file in chunks so it works correctly for large files.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
