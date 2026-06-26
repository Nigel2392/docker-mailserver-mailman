package mailmgmt

import "regexp"

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var stripAnsiRe = regexp.MustCompile(ansi)

func stripAnsi(str []byte) []byte {
	return stripAnsiRe.ReplaceAll(str, []byte(""))
}
