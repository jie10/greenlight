package domain

import "errors"

var (
	ErrNotFound       = errors.New("domain: record not found")
	ErrEditConflict   = errors.New("domain: edit conflict")
	ErrDuplicateEmail = errors.New("domain: duplicate email")
)

type ValidateErrors map[string]string

func (v ValidateErrors) Error() string {
	return "domain: validation failed"
}
