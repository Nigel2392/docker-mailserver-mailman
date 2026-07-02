package errors

import (
	"errors"
	"fmt"
)

type ErrorCode int64

const (
	CodeUnknown ErrorCode = iota << 1
	CodeNotRunning

	CodeAny = 0
)

type MailserverError struct {
	Code    ErrorCode
	pretext string
	cause   error
}

func MailServerError(code ErrorCode) MailserverError {
	return MailserverError{
		Code: code,
	}
}

func IsMailserverError(err error) bool {
	var chk = MailserverError{
		Code: CodeAny,
	}
	return errors.Is(err, chk)
}

func (m MailserverError) Wrap(pretext string) MailserverError {
	if m.pretext != "" {
		pretext = fmt.Sprintf(
			"%s: %s", pretext, m.pretext,
		)
	}

	return MailserverError{
		Code:    m.Code,
		pretext: pretext,
		cause:   m.cause,
	}
}

func (m MailserverError) Wrapf(pretextFmt string, args ...any) MailserverError {
	var pretext = fmt.Sprintf(pretextFmt, args...)

	if m.pretext != "" {
		pretext = fmt.Sprintf(
			"%s: %s", pretext, m.pretext,
		)
	}

	return MailserverError{
		Code:    m.Code,
		pretext: pretext,
		cause:   m.cause,
	}

}
func (m MailserverError) Cause(err error) MailserverError {
	return MailserverError{
		Code:    m.Code,
		pretext: m.pretext,
		cause:   errors.Join(m.cause, err),
	}
}

func (m MailserverError) Error() string {
	if m.cause == nil && m.pretext != "" {
		return m.pretext
	}
	if m.cause == nil && m.pretext == "" {
		return "Unknown error occurred."
	}

	var e = m.cause.Error()
	if m.pretext == "" {
		return e
	}

	// Allocate exact buffer size: len(pretext) + len(": ") + len(cause)
	var l = len(m.pretext) + 2 + len(e)
	var b = make([]byte, l)

	// Keep track of the offset
	offset := copy(b, m.pretext)
	offset += copy(b[offset:], ": ")
	copy(b[offset:], e)

	return string(b)
}

func (m MailserverError) Is(other error) bool {
	chk, ok := other.(MailserverError)
	if !ok {
		return false
	}

	return m.Code == chk.Code || m.Code == CodeAny || chk.Code == CodeAny
}

func (m MailserverError) Unwrap() error {
	return m.cause
}
