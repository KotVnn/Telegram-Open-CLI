package opencode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
)

// Server manages the lifecycle of the `opencode serve` subprocess.
type Server struct {
	command  string
	hostname string
	port     int
	dir      string
	password string
	env      []string

	mu     sync.Mutex
	cmd    *exec.Cmd
	logger zerolog.Logger
}

// NewServer creates a server manager for the given hostname/port.
func NewServer(command, hostname string, port int, dir, password string, env []string, logger zerolog.Logger) *Server {
	if command == "" {
		command = "opencode"
	}
	return &Server{
		command:  command,
		hostname: hostname,
		port:     port,
		dir:      dir,
		password: password,
		env:      env,
		logger:   logger,
	}
}

// Addr returns the base URL the server listens on.
func (s *Server) Addr() string {
	return fmt.Sprintf("http://%s:%d", s.hostname, s.port)
}

// Start spawns the server and waits until it is ready.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		if err := s.cmd.Process.Signal(syscall.Signal(0)); err == nil {
			return nil
		}
	}

	args := []string{"serve", "--port", fmt.Sprintf("%d", s.port), "--hostname", s.hostname}
	cmd := exec.CommandContext(ctx, s.command, args...)
	cmd.Env = append(os.Environ(), s.env...)
	if s.password != "" {
		cmd.Env = append(cmd.Env, "OPENCODE_SERVER_PASSWORD="+s.password)
	}
	if s.dir != "" {
		cmd.Dir = s.dir
	}
	cmd.Stdout = &logWriter{logger: s.logger}
	cmd.Stderr = &logWriter{logger: s.logger}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s serve: %w", s.command, err)
	}

	s.cmd = cmd
	s.logger.Info().Str("addr", s.Addr()).Msg("opencode serve process started")
	return nil
}

// Stop terminates the server process, killing its whole process group.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	cmd := s.cmd
	s.cmd = nil
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	s.logger.Info().Int("pid", cmd.Process.Pid).Msg("stopping opencode serve process")

	// Kill the process group so child processes are cleaned up too.
	if cmd.Process.Pid > 0 {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		if cmd.Process.Pid > 0 {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return ctx.Err()
	case <-done:
		return nil
	}
}

// Running reports whether the managed process is still alive.
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}
	return s.cmd.Process.Signal(syscall.Signal(0)) == nil
}

// logWriter adapts a zerolog logger to an io.Writer for subprocess output.
type logWriter struct {
	logger zerolog.Logger
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.logger.Debug().Msg(string(p))
	return len(p), nil
}
