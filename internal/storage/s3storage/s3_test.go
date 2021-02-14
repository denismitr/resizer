package s3storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_isValidKey(t *testing.T) {
	// valid keys consist of image ID, transformation request and supported extension
	validKeys := []string{"6028336099d807ec425eeed2/h600_w360.png", "6028336099d807ec425eeed2/h600_w360_fit.jpg"}
	// invalid keys are anything that does not match all above criteria
	invalidKeys := []string{"h600_w360_fit.jpg", "foo", ""}

	for _, k := range validKeys {
		t.Run(fmt.Sprintf("valid key %s", k), func(t *testing.T) {
			assert.True(t, isValidKey(k))
		})
	}

	for _, k := range invalidKeys {
		t.Run(fmt.Sprintf("invalid key %s", k), func(t *testing.T) {
			assert.False(t, isValidKey(k))
		})
	}
}
