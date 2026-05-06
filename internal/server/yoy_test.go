package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseYearsParam(t *testing.T) {
	assert.Equal(t, 1, parseYearsParam(""))
	assert.Equal(t, 1, parseYearsParam("abc"))
	assert.Equal(t, 1, parseYearsParam("0"))
	assert.Equal(t, 2, parseYearsParam("2"))
	assert.Equal(t, 10, parseYearsParam("50"))
}
