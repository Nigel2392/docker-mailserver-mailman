package mailmgmt

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	merrs "github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt/errors"
	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt/shell"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

var ErrCommandFailed = errors.New("command failed")

type Command struct {
	s   SetupCommand
	err error
}

func NewCommand(args ...string) *Command {
	return NewCommandCtx(context.Background(), args...)
}

func NewCommandCtx(ctx context.Context, args ...string) *Command {
	return &Command{
		s: SetupCommand{
			ctx:  ctx,
			args: args,
			c:    CONFIG,
		},
	}
}

func (c *Command) String() string {
	return c.s.String()
}

func (c *Command) Error() error {
	return c.err
}

type SetupCommand struct {
	ctx  context.Context
	c    *MailManagementConfig
	args []string
}

func (s SetupCommand) String() string {
	return strings.Join(s.args, " ")
}

func (s *SetupCommand) Arg(args ...string) SetupCommand {
	var newArgs = slices.Clone(s.args)
	newArgs = append(newArgs, args...)
	return SetupCommand{
		ctx:  s.ctx,
		c:    s.c,
		args: newArgs,
	}
}

func (s SetupCommand) Email() MailCommand {
	return MailCommand{
		s: s.Arg("email"),
	}
}

func (s SetupCommand) Alias() AliasCommand {
	return AliasCommand{
		s: s.Arg("alias"),
	}
}

func (s SetupCommand) Quota() QuotaCommand {
	return QuotaCommand{
		s: s.Arg("quota"),
	}
}

func (s SetupCommand) Restrict() RestrictMailCommand {
	return RestrictMailCommand{
		s: s.Arg("email", "restrict"),
	}
}

type ColorStrippingWriter struct {
	strings.Builder
}

func (w *ColorStrippingWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	_, err = w.Builder.Write(stripAnsi(p))
	return n, err
}

func (c Command) Exec2() (out, errOut string, err error) {
	if c.err != nil {
		return "", "", c.err
	}

	logger.Infof("Executing command in pool: %q", c.String())

	resultChan, err := shell.ExecInPool(c.s.ctx, c.String())
	if err != nil {
		return "", "", err
	}

	var result shell.Result
	select {
	case result = <-resultChan:
		// Result received successfully
	case <-c.s.ctx.Done():
		// The HTTP request timed out or the client disconnected
		return "", "", fmt.Errorf("request cancelled while waiting for pool: %w", c.s.ctx.Err())
	}

	cleanOutput := string(stripAnsi([]byte(result.Output)))

	if result.ExitCode != 0 {
		var errs = make([]error, 0)
		var errsList = strings.Split(cleanOutput, "\n")

		for _, e := range errsList {
			var errIdx = strings.Index(e, "ERROR")
			if errIdx < 0 {
				continue
			}
			errs = append(errs, errors.New(e[errIdx:]))
		}

		if len(errs) > 1 {
			// We return cleanOutput for both out and errOut since they are merged
			return cleanOutput, cleanOutput, errors.Join(append([]error{ErrCommandFailed}, errs...)...)
		}

		return cleanOutput, cleanOutput, errors.Join(ErrCommandFailed, fmt.Errorf("(exit code %d)", result.ExitCode), result.Error)
	}

	// If there was a stream death or write error reported by the pool
	if result.Error != nil {
		return cleanOutput, cleanOutput, result.Error
	}

	// Success
	return cleanOutput, "", nil
}

func (c Command) Exec() (out, errOut string, err error) {
	if c.err != nil {
		return "", "", err
	}

	mailserver, err := c.s.c.InspectDockerMailServer(c.s.ctx, false)
	if err != nil {
		return "", "", err
	}

	logger.Infof("Executing command: %q", c.String())

	if !mailserver.Container.State.Running {
		return "", "", merrs.ErrNotRunning.Wrapf("container %q is not running.", mailserver.Container.ID)
	}

	execStart, err := c.s.c.Docker.ExecCreate(
		c.s.ctx,
		mailserver.Container.ID,
		client.ExecCreateOptions{
			Cmd:          []string{"bash", "-c", strings.Join(c.s.args, " ")},
			AttachStdout: true,
			AttachStderr: true,
		},
	)
	if err != nil {
		return "", "", err
	}

	resp, err := c.s.c.Docker.ExecAttach(
		c.s.ctx, execStart.ID, client.ExecAttachOptions{},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	var outBuf, errBuf ColorStrippingWriter
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to read exec output: %w", err)
	}

	inspectResp, err := c.s.c.Docker.ExecInspect(c.s.ctx, execStart.ID, client.ExecInspectOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to inspect exec: %w", err)
	}

	outStr := outBuf.String()
	errStr := errBuf.String()

	if inspectResp.ExitCode != 0 {
		var errs = make([]error, 0)
		var errsList = strings.Split(errStr, "\n")
		for _, e := range errsList {
			var errIdx = strings.Index(e, "ERROR")
			if errIdx < 0 {
				continue
			}

			errs = append(
				errs,
				errors.New(e[errIdx:]),
			)
		}

		if len(errs) > 1 {
			return "", "", errors.Join(append([]error{ErrCommandFailed}, errs...)...)
		}

		return "", "", errors.Join(ErrCommandFailed, fmt.Errorf("(exit code %d)", inspectResp.ExitCode))
	}

	return outStr, errStr, nil
}
