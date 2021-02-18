package backoffice

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_intFromQueryStringOrDefault(t *testing.T) {
	validInputs := []struct{
		input string
		def int
		result int
	}{
		{input: "3", def: 1, result: 3},
		{input: "", def: 1, result: 1},
		{input: "", def: 113, result: 113},
		{input: "4", def: 113, result: 4},
	}

	invalidInputs := []struct{
		input string
		def int
	}{
		{input: "fff", def: 1},
		{input: "-3.4", def: 1},
	}

	for _, tc := range validInputs {
		t.Run(fmt.Sprintf("valid input %s", tc.input), func(t *testing.T) {
			result, err := intFromQueryStringOrDefault(tc.input, tc.def)
			assert.NoError(t, err)
			assert.Equal(t, tc.result, result)
		})
	}

	for _, tc := range invalidInputs {
		t.Run(fmt.Sprintf("invalid input %s", tc.input), func(t *testing.T) {
			result, err := intFromQueryStringOrDefault(tc.input, tc.def)
			assert.Error(t, err)
			assert.Equal(t, 0, result)
		})
	}
}
