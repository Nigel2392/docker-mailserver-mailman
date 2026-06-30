package mailmgmt

var (
	ErrUnknown    = MailServerError(CodeUnknown)
	ErrNotRunning = MailServerError(CodeNotRunning)
)
