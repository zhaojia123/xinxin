package apperror

import (
	"fmt"
	"runtime"
	"strings"
)

type Error struct {
	File    string
	Line    int
	Message string
	Cause   error
}

func New(message string) error {
	_, file, line, _ := runtime.Caller(1)
	return &Error{File: projectPath(file), Line: line, Message: message}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	_, file, line, _ := runtime.Caller(1)
	return &Error{File: projectPath(file), Line: line, Message: message, Cause: err}
}

func At(file string, line int, message string) error {
	return &Error{File: projectPath(file), Line: line, Message: message}
}

func (e *Error) Error() string {
	location := fmt.Sprintf("%s:%d", e.File, e.Line)
	if e.Cause != nil {
		return fmt.Sprintf("%s：%s；原因：%v", location, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s：%s", location, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func projectPath(file string) string {
	if index := strings.LastIndex(file, "/friends-records/"); index >= 0 {
		return strings.TrimPrefix(file[index+len("/friends-records/"):], "/")
	}
	for _, marker := range []string{"/cmd/", "/internal/", "/webassets/"} {
		if index := strings.LastIndex(file, marker); index >= 0 {
			return strings.TrimPrefix(file[index:], "/")
		}
	}
	return file
}
