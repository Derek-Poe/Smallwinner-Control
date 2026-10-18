package control

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

// Action is deliberately typed. There are no program names, shell fragments,
// paths, arbitrary arguments, or device addresses in the public action surface.
type Action struct {
	Kind      string
	Resources Resources
}

const ApplyResources = "apply_resources"
const MaxOutput = 1024

var ErrOutputLimit = errors.New("command output limit exceeded")

// Runner implementations are trusted and MUST honor ctx. No goroutine detaches
// a timed-out command: its caller retains ownership until Run returns.
type Runner interface {
	Run(context.Context, Action, io.Writer) error
}
type cappedWriter struct {
	n        int
	exceeded bool
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	if len(p) > MaxOutput-w.n {
		w.exceeded = true
		return 0, ErrOutputLimit
	}
	w.n += len(p)
	return len(p), nil
}
func Execute(ctx context.Context, runner Runner, action Action, timeout time.Duration) error {
	if action.Kind != ApplyResources {
		return errors.New("action not allowed")
	}
	if err := action.Resources.Validate(); err != nil {
		return err
	}
	if timeout <= 0 || timeout > 5*time.Second {
		return errors.New("invalid command deadline")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	output := &cappedWriter{}
	err := runner.Run(ctx, action, output)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if output.exceeded {
		return ErrOutputLimit
	}
	return err
}

// Simulator never creates a process or contacts a device. Process state is
// deterministic illustrative data, not host telemetry.
type Simulator struct {
	mu    sync.Mutex
	state DeviceState
}

func NewSimulator() *Simulator {
	s := &Simulator{state: DeviceState{Resources: Resources{CPU: "balanced", Density: 240}, Apps: append([]App{}, Catalog...)}}
	_ = s.Observe(Observation{Foreground: "demo.home", Temperature: 35})
	return s
}
func (s *Simulator) Observe(o Observation) error {
	if err := o.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Observation = o
	s.state.Processes = []Process{{PID: 101, Package: "demo.home", RSSMiB: 24}}
	if o.Foreground == "demo.reader" {
		s.state.Processes = append(s.state.Processes, Process{PID: 102, Package: o.Foreground, RSSMiB: 48})
	}
	if o.Foreground == "demo.game" {
		s.state.Processes = append(s.state.Processes, Process{PID: 103, Package: o.Foreground, RSSMiB: 160})
	}
	return nil
}
func (s *Simulator) Snapshot() DeviceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	st.Apps = append([]App{}, st.Apps...)
	st.Processes = append([]Process{}, st.Processes...)
	return st
}
func (s *Simulator) Run(ctx context.Context, a Action, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Kind != ApplyResources {
		return errors.New("unsupported simulated action")
	}
	if err := a.Resources.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "simulated resources applied\n"); err != nil {
		return err
	}
	s.state.Resources = a.Resources
	return nil
}
