package utils

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidDateRange   = errors.New("start date must be before or equal to end date")
	ErrPastDate           = errors.New("cannot apply for leave in the past")
	ErrOverlappingLeave   = errors.New("overlapping leave request exists")
	ErrInsufficientBalance = errors.New("insufficient leave balance")
	ErrLeaveNotFound      = errors.New("leave not found")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrInvalidLeaveType   = errors.New("invalid leave type")
)

