package domain

import "errors"

var (
	// ErrDuplicateEvent signals that the webhook event was already processed.
	ErrDuplicateEvent = errors.New("duplicate event")

	// ErrInvalidCPF signals that a provided CPF is malformed or fails checksum validation.
	ErrInvalidCPF = errors.New("invalid cpf")

	// ErrNotFound signals that a notification was not found or is not owned by the caller.
	ErrNotFound = errors.New("notification not found")

	// ErrAlreadyRead signals that a notification is already marked as read.
	ErrAlreadyRead = errors.New("notification already read")
)
