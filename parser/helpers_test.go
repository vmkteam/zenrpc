package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadPackage(t *testing.T) {
	_, err := loadPackage("../testdata/subservice/subarithservice.go")
	assert.NoError(t, err)
}
