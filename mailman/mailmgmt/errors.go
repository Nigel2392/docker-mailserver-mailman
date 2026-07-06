package mailmgmt

import "github.com/Nigel2392/errors"

const (
	CodeDockerError errors.GoCode = "DockerError"
)

var (
	ErrDocker = errors.New(CodeDockerError, "error while interacting with docker client")
)
