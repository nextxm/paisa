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
	return SHA256Files([]string{path})
}

// SHA256Files computes the SHA-256 hex digest of the combined contents of the
// files at the given paths. It streams the files in chunks and hashes them in
// the order provided.
func SHA256Files(paths []string) (string, error) {
	h := sha256.New()
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
