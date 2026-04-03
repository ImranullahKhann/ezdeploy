package project

import "errors"

var (
	ErrNotFound       = errors.New("project not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidInput   = errors.New("invalid input")
	ErrAlreadyExists  = errors.New("project already exists")
	ErrConfigNotFound = errors.New("project config not found")
)
