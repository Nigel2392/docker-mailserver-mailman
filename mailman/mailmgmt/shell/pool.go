package shell

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/google/uuid"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

// Task represents a command to be executed in the pool
type Task struct {
	ID         string
	Command    string
	ResultChan chan Result
}

// Result holds the output and exit status of a Task
type Result struct {
	Output   string
	ExitCode int
	Error    error
}

type execPoolKey struct{}

type ExecPool struct {
	mu       sync.Mutex
	resp     *client.HijackedResponse
	cancel   context.CancelFunc
	taskChan chan Task
}

func ContextWithExecPool(ctx context.Context, p *ExecPool) context.Context {
	return context.WithValue(ctx, execPoolKey{}, p)
}

func ExecPoolFromContext(ctx context.Context) (*ExecPool, bool) {
	var p, ok = ctx.Value(execPoolKey{}).(*ExecPool)
	return p, ok
}

// StartPool initializes the session and starts the background worker.
func StartPool(ctx context.Context, cli *client.Client, containerId string) (context.Context, *ExecPool, error) {
	execStart, err := cli.ExecCreate(ctx, containerId, client.ExecCreateOptions{
		Cmd:          []string{"bash"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          false,
	})
	if err != nil {
		return ctx, nil, err
	}

	resp, err := cli.ExecAttach(ctx, execStart.ID, client.ExecAttachOptions{})
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to attach to pool exec: %w", err)
	}

	poolCtx, cancel := context.WithCancel(ctx)
	pool := &ExecPool{
		resp:     &resp.HijackedResponse,
		cancel:   cancel,
		taskChan: make(chan Task, 100),
	}

	go pool.worker()

	return ContextWithExecPool(poolCtx, pool), pool, nil
}

// ExecInPool executes a command in the context of an existing container. The output is sent over the ResultChan.
func ExecInPool(ctx context.Context, cmd string) (chan Result, error) {
	var pool, ok = ExecPoolFromContext(ctx)
	if !ok || pool == nil {
		return nil, errors.New("no exec pool found")
	}

	resultChan := make(chan Result, 1)
	pool.taskChan <- Task{
		ID:         uuid.New().String(),
		Command:    cmd,
		ResultChan: resultChan,
	}

	return resultChan, nil
}

// worker sequentially processes tasks to prevent stream corruption
func (p *ExecPool) worker() {
	defer p.resp.Close()

	// process the output from the worker
	r, w := io.Pipe()
	go func() {
		_, err := stdcopy.StdCopy(w, w, p.resp.Reader)
		if err != nil {
			logger.Errorf("Error while reading from docker cotnainer: %v", err)
		}
		w.Close()
	}()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for task := range p.taskChan {
		delimiter := fmt.Sprintf("---DONE-%s---", task.ID)

		payload := fmt.Sprintf("%s ; echo \"%s:$?\"\n", task.Command, delimiter)
		_, err := p.resp.Conn.Write([]byte(payload))

		if err != nil {
			task.ResultChan <- Result{Error: fmt.Errorf("write failed: %w", err)}
			continue
		}

		var outputBuilder strings.Builder
		var exitCode int
		var streamErr error // Track underlying stream death

		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, delimiter) {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &exitCode)
				}
				break
			}

			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")
		}

		if err := scanner.Err(); err != nil {
			streamErr = fmt.Errorf("stream read error (container likely died): %w", err)
		}

		var finalErr error
		if streamErr != nil {
			finalErr = streamErr
		} else if exitCode != 0 {
			finalErr = fmt.Errorf("command failed with exit code %d", exitCode)
		}

		task.ResultChan <- Result{
			Output:   outputBuilder.String(),
			ExitCode: exitCode,
			Error:    finalErr,
		}

		// If the stream is broken, there is no point processing further tasks.
		// Break the worker loop and let the defer close everything down.
		if streamErr != nil {
			fmt.Printf("Worker pool shutting down due to stream error: %v\n", streamErr)
			break
		}
	}
}

// Execute wraps the channel interaction for the rest of your app
func (p *ExecPool) Execute(cmd string) Result {
	resultChan := make(chan Result, 1)
	p.taskChan <- Task{
		ID:         uuid.New().String(),
		Command:    cmd,
		ResultChan: resultChan,
	}
	return <-resultChan
}
