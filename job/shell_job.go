package job

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/reugn/go-quartz/quartz"
)

// ShellJob represents a shell command Job, implements the [quartz.Job] interface.
type ShellJob struct {
	mtx       sync.RWMutex // Use RWMutex for better concurrent performance
	cmd       string
	timeout   time.Duration
	result    *ShellJobResult
	jobStatus Status
	callback  func(context.Context, *ShellJob)
}

type ShellJobResult struct {
	exitCode int
	stdout   string
	stderr   string
}

var _ quartz.Job = (*ShellJob)(nil)

// NewShellJob returns a new [ShellJob] for the given command.
func NewShellJob(cmd string, opts ...ShellJobOptionFunc) *ShellJob {
	if cmd == "" {
		panic("command cannot be empty")
	}

	job := &ShellJob{
		cmd:       cmd,
		jobStatus: StatusNA,
		timeout:   JobTimeout,
		result:    &ShellJobResult{}, // Initialize to avoid nil pointer
	}

	for _, opt := range opts {
		opt(job)
	}
	return job
}

type ShellJobOptionFunc func(job *ShellJob)

func WithTimeout(timeout time.Duration) ShellJobOptionFunc {
	return func(job *ShellJob) {
		if timeout > 0 {
			job.timeout = timeout
		}
	}
}

func WithCallback(callback func(ctx context.Context, job *ShellJob)) ShellJobOptionFunc {
	return func(job *ShellJob) {
		job.callback = callback
	}
}

// Description returns the description of the ShellJob.
func (sh *ShellJob) Description() string {
	return fmt.Sprintf("ShellJob%s%s", quartz.Sep, sh.cmd)
}

var (
	shellOnce = sync.Once{}
	shellPath = "bash"
)

func getShell() string {
	shellOnce.Do(func() {
		_, err := exec.LookPath("/bin/bash")
		if err != nil {
			shellPath = "sh"
		}
	})
	return shellPath
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (sh *ShellJob) Execute(ctx context.Context) error {
	shell := getShell()
	var stdout, stderr bytes.Buffer

	// Set timeout context if specified
	var cancel context.CancelFunc
	if sh.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, sh.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, shell, "-c", sh.cmd)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set process group for signal propagation
	if sh.timeout > 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// Start the command
	err := cmd.Start()
	if err != nil {
		sh.setJobResult(stdout.String(), stderr.String(), -1, StatusFailure)
		return err
	}

	// Wait for command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var finalErr error

	select {
	case err = <-done:
		// Command completed normally (success or failure)
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		} else if err != nil {
			exitCode = -1
		}

		status := StatusOK
		if err != nil {
			status = StatusFailure
		}

		sh.setJobResult(stdout.String(), stderr.String(), exitCode, status)
		finalErr = err

	case <-ctx.Done():
		// Timeout handling
		sh.terminateProcess(cmd)
		<-done // Wait for goroutine to complete

		sh.setJobResult(stdout.String(), stderr.String(), -1, StatusTimeout)
		finalErr = ctx.Err()
	}

	// Execute callback if provided
	// TODO: Use an event system (e.g., JOB_ADD, JOB_FAILED) instead of a direct callback.
	if sh.callback != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't affect main flow
					fmt.Printf("Callback panic: %v\n", r)
				}
			}()
			sh.callback(ctx, sh)
		}()
	}

	return finalErr
}

// setJobResult is a helper method to set job result uniformly
func (sh *ShellJob) setJobResult(stdout, stderr string, exitCode int, status Status) {
	sh.mtx.Lock()
	defer sh.mtx.Unlock()

	sh.result.stdout = stdout
	sh.result.stderr = stderr
	sh.result.exitCode = exitCode
	sh.jobStatus = status
}

// terminateProcess is a helper method to terminate process
func (sh *ShellJob) terminateProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Try graceful termination first
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		// If TERM signal fails, send KILL directly
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return
	}

	// Wait briefly for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Force termination
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func (sh *ShellJob) ExitCode() int {
	sh.mtx.RLock()
	defer sh.mtx.RUnlock()
	return sh.result.exitCode
}

func (sh *ShellJob) Result() *ShellJobResult {
	sh.mtx.RLock()
	defer sh.mtx.RUnlock()
	// Return copy to prevent external modification
	return &ShellJobResult{
		exitCode: sh.result.exitCode,
		stdout:   sh.result.stdout,
		stderr:   sh.result.stderr,
	}
}

func (sh *ShellJob) Stdout() string {
	sh.mtx.RLock()
	defer sh.mtx.RUnlock()
	return sh.result.stdout
}

func (sh *ShellJob) Stderr() string {
	sh.mtx.RLock()
	defer sh.mtx.RUnlock()
	return sh.result.stderr
}

func (sh *ShellJob) JobStatus() Status {
	sh.mtx.RLock()
	defer sh.mtx.RUnlock()
	return sh.jobStatus
}
