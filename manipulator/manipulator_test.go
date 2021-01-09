package manipulator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResult_OriginalFilename(t *testing.T) {
	r1 := Result{Height: 24, Width: 45, Extension: string(PNG)}
	assert.Equal(t, "h24_w45.png", r1.OriginalFilename())
}