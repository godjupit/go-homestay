package platform

import (
	"errors"
	"fmt"
)

const (
	CodeCommon    uint32 = 100001
	CodeParam     uint32 = 100002
	CodeToken     uint32 = 100003
	CodeForbidden uint32 = 100004
	CodeDB        uint32 = 100005
)

type AppError struct {
	Code    uint32
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}
func (e *AppError) Unwrap() error { return e.Cause }
func E(code uint32, msg string, cause error) error {
	return &AppError{Code: code, Message: msg, Cause: cause}
}
func Public(err error) (uint32, string) {
	var app *AppError
	if errors.As(err, &app) {
		return app.Code, app.Message
	}
	return CodeCommon, "服务器开小差啦，稍后再来试一试"
}
