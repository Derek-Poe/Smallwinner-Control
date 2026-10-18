package control

import (
	"context"
	"sync"
	"time"
)

type Controller struct {
	mu      sync.Mutex
	store   *Store
	device  *Simulator
	runner  Runner
	state   State
	timeout time.Duration
}

func NewController(store *Store, device *Simulator) *Controller {
	baseline := device.Snapshot().Resources
	return &Controller{store: store, device: device, runner: device, timeout: 250 * time.Millisecond,
		state: State{Mode: "simulation", Baseline: baseline, Desired: baseline, Policy: "baseline", Revision: store.Snapshot().Revision}}
}
func (c *Controller) Snapshot() State { c.mu.Lock(); defer c.mu.Unlock(); return c.snapshot() }
func (c *Controller) snapshot() State { st := c.state; st.Device = c.device.Snapshot(); return st }
func (c *Controller) Config() Config  { c.mu.Lock(); defer c.mu.Unlock(); return c.store.Snapshot() }
func (c *Controller) Observe(ctx context.Context, o Observation) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.device.Observe(o); err != nil {
		return c.snapshot(), err
	}
	return c.apply(ctx, false)
}
func (c *Controller) Update(ctx context.Context, cfg Config) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.store.Update(cfg); err != nil {
		return c.snapshot(), err
	}
	return c.apply(ctx, false)
}
func (c *Controller) Restore(ctx context.Context) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.apply(ctx, true)
}
func (c *Controller) apply(ctx context.Context, restore bool) (State, error) {
	cfg := c.store.Snapshot()
	observed := c.device.Snapshot()
	target, policy, throttled := Target(cfg, observed.Observation, c.state.Baseline, c.state.Throttled)
	if restore {
		target, policy, throttled = c.state.Baseline, "restored", false
	}
	c.state.Revision, c.state.Desired, c.state.Throttled = cfg.Revision, target, throttled
	c.state.Error = ""
	if observed.Resources != target {
		c.state.Commands++
		if err := Execute(ctx, c.runner, Action{Kind: ApplyResources, Resources: target}, c.timeout); err != nil {
			c.state.Error = "resource application failed; inspect observed state before retrying"
			return c.snapshot(), err
		}
	}
	c.state.Policy = policy
	return c.snapshot(), nil
}
