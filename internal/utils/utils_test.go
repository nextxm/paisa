package utils

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSubPath(t *testing.T) {
	path, err := BuildSubPath(filepath.FromSlash("/usr/home/john/paisa"), "main.ledger")
	assert.Nil(t, err)
	assert.Equal(t, filepath.FromSlash("/usr/home/john/paisa/main.ledger"), path)

	path, err = BuildSubPath(filepath.FromSlash("/usr/home/john/paisa"), "subfolder/main.ledger")
	assert.Nil(t, err)
	assert.Equal(t, filepath.FromSlash("/usr/home/john/paisa/subfolder/main.ledger"), path)

	path, err = BuildSubPath(filepath.FromSlash("/usr/home/john/paisa"), "../../../subfolder/travel.ledger")
	assert.Error(t, err)

	path, err = BuildSubPath(filepath.FromSlash("/usr/home/john/paisa"), "..")
	assert.Error(t, err)

	path, err = BuildSubPath(filepath.FromSlash("/usr/home/john/paisa"), "./..")
	assert.Error(t, err)

	path, err = BuildSubPath(filepath.FromSlash("/usr/home/john/paisa"), "./../test.ledger")
	assert.Error(t, err)
}
