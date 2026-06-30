package mailmgmt

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

type mailserverError struct {
	code    ErrorCode
	pretext string
	cause   error
}

func MailServerError(code ErrorCode) mailserverError {
	return mailserverError{
		code: code,
	}
}

func IsMailserverError(err error) bool {
	var chk = mailserverError{
		code: CodeAny,
	}
	return errors.Is(err, chk)
}

func (m mailserverError) Wrap(pretext string) mailserverError {
	if m.pretext != "" {
		pretext = fmt.Sprintf(
			"%s: %s", pretext, m.pretext,
		)
	}

	return mailserverError{
		code:    m.code,
		pretext: pretext,
		cause:   m.cause,
	}
}

func (m mailserverError) Wrapf(pretextFmt string, args ...any) mailserverError {
	var pretext = fmt.Sprintf(pretextFmt, args...)

	if m.pretext != "" {
		pretext = fmt.Sprintf(
			"%s: %s", pretext, m.pretext,
		)
	}

	return mailserverError{
		code:    m.code,
		pretext: pretext,
		cause:   m.cause,
	}

}
func (m mailserverError) Cause(err error) mailserverError {
	return mailserverError{
		code:    m.code,
		pretext: m.pretext,
		cause:   errors.Join(m.cause, err),
	}
}

func (m mailserverError) Error() string {
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

func (m mailserverError) Is(other error) bool {
	chk, ok := other.(mailserverError)
	if !ok {
		return false
	}

	return m.code == chk.code || m.code == CodeAny || chk.code == CodeAny
}

func (m mailserverError) Unwrap() error {
	return m.cause
}
