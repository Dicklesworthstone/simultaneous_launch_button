package testutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
)

// TruePath returns the platform-appropriate path to the 'true' command.
// On macOS, 'true' is at /usr/bin/true; on Linux, it's at /bin/true.
func TruePath() string {
	if runtime.GOOS == "darwin" {
		return "/usr/bin/true"
	}
	return "/bin/true"
}

// FalsePath returns the platform-appropriate path to the 'false' command.
// On macOS, 'false' is at /usr/bin/false; on Linux, it's at /bin/false.
func FalsePath() string {
	if runtime.GOOS == "darwin" {
		return "/usr/bin/false"
	}
	return "/bin/false"
}

// CommandExecutor abstracts command execution for testing.
// Production code can use RealExecutor, while tests can use MockExecutor.
type CommandExecutor interface {
	// Run executes a command and returns its combined output.
	Run(name string, args ...string) ([]byte, error)

	// Start begins executing a command without waiting for it to complete.
	// Returns the underlying Cmd for control (Wait, Kill, etc).
	Start(name string, args ...string) (*exec.Cmd, error)

	// RunWithInput executes a command with stdin input.
	RunWithInput(input []byte, name string, args ...string) ([]byte, error)
}

// RealExecutor uses actual exec.Command for production use.
type RealExecutor struct{}

// Run executes a command and returns combined stdout/stderr.
func (r RealExecutor) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Start begins a command without waiting.
func (r RealExecutor) Start(name string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	return cmd, cmd.Start()
}

// RunWithInput executes a command with provided stdin.
func (r RealExecutor) RunWithInput(input []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.CombinedOutput()
}

// CommandCall records a single command invocation.
type CommandCall struct {
	Name  string
	Args  []string
	Input []byte // stdin input if RunWithInput was used
}

// MockExecutor records and simulates command execution for testing.
type MockExecutor struct {
	mu sync.Mutex

	// RecordedCalls contains all commands that were invoked.
	RecordedCalls []CommandCall

	// MockOutput is returned by Run and RunWithInput.
	MockOutput []byte

	// MockError is returned by all methods.
	MockError error

	// OutputFunc allows dynamic output based on the command.
	// If set, this is called instead of returning MockOutput.
	OutputFunc func(name string, args []string) ([]byte, error)
}

// NewMockExecutor creates a mock with configurable static behavior.
func NewMockExecutor(output []byte, err error) *MockExecutor {
	return &MockExecutor{MockOutput: output, MockError: err}
}

// NewMockExecutorFunc creates a mock with dynamic behavior based on command.
func NewMockExecutorFunc(fn func(name string, args []string) ([]byte, error)) *MockExecutor {
	return &MockExecutor{OutputFunc: fn}
}

// Run records the call and returns configured output/error.
func (m *MockExecutor) Run(name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	m.RecordedCalls = append(m.RecordedCalls, CommandCall{Name: name, Args: args})
	m.mu.Unlock()

	if m.OutputFunc != nil {
		return m.OutputFunc(name, args)
	}
	return m.MockOutput, m.MockError
}

// Start records the call and returns nil cmd with configured error.
// Note: The returned *exec.Cmd is nil since we're mocking.
func (m *MockExecutor) Start(name string, args ...string) (*exec.Cmd, error) {
	m.mu.Lock()
	m.RecordedCalls = append(m.RecordedCalls, CommandCall{Name: name, Args: args})
	m.mu.Unlock()

	return nil, m.MockError
}

// RunWithInput records the call including input and returns configured output/error.
func (m *MockExecutor) RunWithInput(input []byte, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	m.RecordedCalls = append(m.RecordedCalls, CommandCall{Name: name, Args: args, Input: input})
	m.mu.Unlock()

	if m.OutputFunc != nil {
		return m.OutputFunc(name, args)
	}
	return m.MockOutput, m.MockError
}

// CallCount returns the number of recorded calls.
func (m *MockExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.RecordedCalls)
}

// LastCall returns the most recent command call, or nil if none.
func (m *MockExecutor) LastCall() *CommandCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.RecordedCalls) == 0 {
		return nil
	}
	// Return a copy to avoid race conditions
	call := m.RecordedCalls[len(m.RecordedCalls)-1]
	return &call
}

// Reset clears all recorded calls.
func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RecordedCalls = nil
}

// WasCalled returns true if any command was invoked.
func (m *MockExecutor) WasCalled() bool {
	return m.CallCount() > 0
}

// WasCalledWith returns true if the specified command was invoked.
func (m *MockExecutor) WasCalledWith(name string, args ...string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.RecordedCalls {
		if call.Name == name && argsMatch(call.Args, args) {
			return true
		}
	}
	return false
}

// argsMatch checks if two arg slices are equal.
func argsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CommandSequenceMock returns different outputs for sequential calls.
// Useful for testing retry logic or multi-step command sequences.
type CommandSequenceMock struct {
	mu       sync.Mutex
	index    int
	Sequence []SequenceStep
}

// SequenceStep defines output for one step in a command sequence.
type SequenceStep struct {
	Output []byte
	Error  error
}

// NewCommandSequenceMock creates a mock that returns different results per call.
func NewCommandSequenceMock(steps ...SequenceStep) *CommandSequenceMock {
	return &CommandSequenceMock{Sequence: steps}
}

// Run returns the next output in the sequence, or an error if exhausted.
func (m *CommandSequenceMock) Run(name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.index >= len(m.Sequence) {
		return nil, fmt.Errorf("mock sequence exhausted after %d calls", len(m.Sequence))
	}
	step := m.Sequence[m.index]
	m.index++
	return step.Output, step.Error
}

// Start returns nil cmd with the next error in sequence.
func (m *CommandSequenceMock) Start(name string, args ...string) (*exec.Cmd, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.index >= len(m.Sequence) {
		return nil, fmt.Errorf("mock sequence exhausted after %d calls", len(m.Sequence))
	}
	step := m.Sequence[m.index]
	m.index++
	return nil, step.Error
}

// RunWithInput returns the next output in the sequence.
func (m *CommandSequenceMock) RunWithInput(input []byte, name string, args ...string) ([]byte, error) {
	return m.Run(name, args...)
}

// Reset resets the sequence to the beginning.
func (m *CommandSequenceMock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.index = 0
}
