package mongosync

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// Manager handles mongosync process management
type Manager struct {
	binaryPath string
	sourceURI  string
	targetURI  string
	logger     *slog.Logger
	cmd        *exec.Cmd
}

func New(binaryPath, sourceURI, targetURI string, logger *slog.Logger) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		sourceURI:  sourceURI,
		targetURI:  targetURI,
		logger:     logger,
	}
}

func (m *Manager) Start() error {
	m.logger.Info("Starting mongosync process...")

	m.cmd = exec.Command(
		m.binaryPath,
		"--acceptDisclaimer",
		"--cluster0", m.sourceURI,
		"--cluster1", m.targetURI,
	)

	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.cmd.Stdin = os.Stdin

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mongosync: %w", err)
	}

	m.logger.Info("Waiting for mongosync to initialize...")
	time.Sleep(5 * time.Second)

	return nil
}

// Wait waits for the mongosync process to complete
func (m *Manager) Wait() error {
	if m.cmd == nil {
		return fmt.Errorf("mongosync process not started")
	}

	m.logger.Info("Waiting for mongosync to complete...")
	if err := m.cmd.Wait(); err != nil {
		return fmt.Errorf("mongosync process failed: %w", err)
	}

	m.logger.Info("Mongosync process completed successfully")
	return nil
}

// Stop stops the mongosync process
func (m *Manager) Stop() error {
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	m.logger.Info("Stopping mongosync process...")
	return m.cmd.Process.Kill()
}
