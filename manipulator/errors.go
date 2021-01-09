package manipulator

import "fmt"

type ValidationError struct {
	errors map[string]string
}

func NewValidationError() *ValidationError {
	return &ValidationError{errors: make(map[string]string)}
}

func (err *ValidationError) Add(k, v string) {
	err.errors[k] = v
}

func (err *ValidationError) Empty() bool {
	return len(err.errors) == 0
}

func (err *ValidationError) Error() string {
	return fmt.Sprint("Validation")
}

func (err *ValidationError) Errors() map[string]string {
	return err.errors
}
