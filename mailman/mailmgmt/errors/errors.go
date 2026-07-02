package errors

var (
	ErrUnknown    = MailServerError(CodeUnknown)
	ErrNotRunning = MailServerError(CodeNotRunning)
)
