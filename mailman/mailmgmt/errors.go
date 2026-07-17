package mailmgmt

import "github.com/Nigel2392/errors"

const (
	CodeAPIError       errors.GoCode = "APIError"
	CodeDockerError    errors.GoCode = "DockerError"
	CodeQuotaError     errors.GoCode = "QuotaError"
	CodeQuotaNotExists errors.GoCode = "QuotaNotExists"
)

var (
	ErrAPI            = errors.New(CodeAPIError, "error while interacting with API")
	ErrDocker         = errors.New(CodeDockerError, "error while interacting with docker client")
	ErrQuota          = errors.New(CodeQuotaError, "error while fetching quota")
	ErrQuotaNotExists = errors.New(CodeQuotaNotExists, "quota does not exist")
)
