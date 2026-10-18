package control

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testController(t *testing.T) (*Controller, *Store) {
	t.Helper()
	s, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return NewController(s, NewSimulator()), s
}
func TestPolicyTransitions(t *testing.T) {
	c, _ := testController(t)
	cases := []struct {
		pkg      string
		temp     float64
		cpu      string
		density  int
		policy   string
		thermal  bool
		commands uint64
	}{
		{"demo.game", 35, "performance", 280, "demo.game", false, 1},
		{"demo.game", 75, "balanced", 240, "thermal", true, 2},
		{"demo.reader", 72, "balanced", 240, "thermal", true, 2},
		{"demo.reader", 70, "balanced", 240, "thermal", true, 2},
		{"demo.reader", 69.9, "powersave", 220, "demo.reader", false, 3},
		{"demo.home", 35, "balanced", 240, "baseline", false, 4},
	}
	for _, tc := range cases {
		t.Run(tc.pkg+"/"+tc.policy, func(t *testing.T) {
			st, err := c.Observe(context.Background(), Observation{tc.pkg, tc.temp})
			if err != nil {
				t.Fatal(err)
			}
			if st.Device.Resources != (Resources{tc.cpu, tc.density}) || st.Policy != tc.policy || st.Throttled != tc.thermal || st.Commands != tc.commands {
				t.Fatalf("unexpected state: %+v", st)
			}
		})
	}
}
func TestRuleUpdateRemovalAndExplicitRestore(t *testing.T) {
	c, _ := testController(t)
	ctx := context.Background()
	_, _ = c.Observe(ctx, Observation{"demo.game", 35})
	cfg := c.Config()
	cfg.Rules[1].Resources.Density = 320
	st, err := c.Update(ctx, cfg)
	if err != nil || st.Device.Resources.Density != 320 {
		t.Fatalf("update: %+v %v", st, err)
	}
	st, err = c.Restore(ctx)
	if err != nil || st.Device.Resources != st.Baseline || st.Policy != "restored" {
		t.Fatal(st, err)
	}
	st, _ = c.Observe(ctx, Observation{"demo.game", 35})
	if st.Device.Resources.Density != 320 {
		t.Fatal("policy did not resume")
	}
	cfg = c.Config()
	cfg.Rules = nil
	st, err = c.Update(ctx, cfg)
	if err != nil || st.Device.Resources != st.Baseline || st.Policy != "baseline" {
		t.Fatal(st, err)
	}
}

type runnerFunc func(context.Context, Action, io.Writer) error

func (f runnerFunc) Run(c context.Context, a Action, w io.Writer) error { return f(c, a, w) }
func TestCommandFailuresAndRetry(t *testing.T) {
	c, _ := testController(t)
	c.runner = runnerFunc(func(context.Context, Action, io.Writer) error { return errors.New("injected") })
	st, err := c.Observe(context.Background(), Observation{"demo.game", 35})
	if err == nil || st.Error == "" || st.Device.Resources != st.Baseline || st.Policy != "baseline" || st.Desired.CPU != "performance" {
		t.Fatal(st, err)
	}
	c.runner = c.device
	st, err = c.Observe(context.Background(), Observation{"demo.game", 35})
	if err != nil || st.Error != "" || st.Policy != "demo.game" || st.Commands != 2 {
		t.Fatal(st, err)
	}
}
func TestBoundedExecutor(t *testing.T) {
	a := Action{ApplyResources, Resources{"balanced", 240}}
	t.Run("timeout retains ownership", func(t *testing.T) {
		returned := false
		runner := runnerFunc(func(ctx context.Context, _ Action, _ io.Writer) error {
			<-ctx.Done()
			returned = true
			return ctx.Err()
		})
		err := Execute(context.Background(), runner, a, 5*time.Millisecond)
		if !errors.Is(err, context.DeadlineExceeded) || !returned {
			t.Fatal(err, returned)
		}
	})
	t.Run("output even when runner ignores write error", func(t *testing.T) {
		runner := runnerFunc(func(_ context.Context, _ Action, w io.Writer) error {
			_, _ = io.WriteString(w, strings.Repeat("x", MaxOutput+1))
			return nil
		})
		if !errors.Is(Execute(context.Background(), runner, a, time.Second), ErrOutputLimit) {
			t.Fatal("output cap not enforced")
		}
	})
	t.Run("reject arbitrary action before runner", func(t *testing.T) {
		runner := runnerFunc(func(context.Context, Action, io.Writer) error { t.Fatal("runner called"); return nil })
		if Execute(context.Background(), runner, Action{"shell", a.Resources}, time.Second) == nil {
			t.Fatal("accepted unknown action")
		}
	})
}
func TestSimulationSnapshotsAreIndependent(t *testing.T) {
	s := NewSimulator()
	before := s.Snapshot()
	before.Apps[0].Label = "changed"
	if s.Snapshot().Apps[0].Label == "changed" {
		t.Fatal("mutable snapshot")
	}
	if err := s.Observe(Observation{"demo.game", 35}); err != nil {
		t.Fatal(err)
	}
	st := s.Snapshot()
	if len(st.Processes) != 2 || st.Processes[1].PID != 103 || st.Processes[1].RSSMiB != 160 {
		t.Fatal(st)
	}
	if s.Observe(Observation{"arbitrary.package", 35}) == nil {
		t.Fatal("unknown app accepted")
	}
}
func TestConcurrentConfigCASAndAtomicReaders(t *testing.T) {
	_, s := testController(t)
	cfg := s.Snapshot()
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Update(cfg)
			if err == nil {
				success.Add(1)
			} else if !errors.Is(err, ErrConflict) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || s.Snapshot().Revision != 2 {
		t.Fatal(success.Load(), s.Snapshot())
	}
	done := make(chan struct{})
	errs := make(chan error, 1)
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			b, e := os.ReadFile(filepath.Join(s.dir, "config.json"))
			if e != nil {
				errs <- e
				return
			}
			var c Config
			if e = json.Unmarshal(b, &c); e != nil {
				errs <- e
				return
			}
			if e = c.Validate(); e != nil {
				errs <- e
				return
			}
		}
	}()
	for i := 0; i < 30; i++ {
		cfg = s.Snapshot()
		cfg.ThermalLimit = 75 + float64(i%2)
		if _, err := s.Update(cfg); err != nil {
			t.Fatal(err)
		}
	}
	<-done
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
	var disk Config
	b, _ := os.ReadFile(filepath.Join(s.dir, "config.json"))
	_ = json.Unmarshal(b, &disk)
	if !reflect.DeepEqual(disk, s.Snapshot()) {
		t.Fatal("disk and memory diverged")
	}
}
func TestPersistenceFailureDoesNotPublishRevision(t *testing.T) {
	_, s := testController(t)
	before := s.Snapshot()
	original, _ := os.ReadFile(filepath.Join(s.dir, "config.json"))
	s.write = func(string, []byte) error { return errors.New("injected disk failure") }
	if _, err := s.Update(before); err == nil {
		t.Fatal("failure not reported")
	}
	after, _ := os.ReadFile(filepath.Join(s.dir, "config.json"))
	if !reflect.DeepEqual(s.Snapshot(), before) || string(original) != string(after) {
		t.Fatal("failed transaction published")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if atomicWrite(dir, []byte("{}")) == nil {
		t.Fatal("rename failure not reported")
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".config-*"))
	if len(leftovers) != 0 {
		t.Fatal("temporary files leaked")
	}
}
func TestStoreOwnershipRestartAndMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if second, e := OpenStore(dir); e == nil {
		second.Close()
		t.Fatal("second owner accepted")
	}
	cfg := s.Snapshot()
	cfg.ThermalLimit = 81
	updated, err := s.Update(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Snapshot(), updated) {
		t.Fatal("restart lost config")
	}
	s.Close()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(dir); err == nil {
		t.Fatal("corrupt config silently replaced")
	}
	if err := os.Remove(filepath.Join(dir, "config.json")); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dir)
	if err != nil {
		t.Fatal("failed constructor leaked lease", err)
	}
	s.Close()
}
func TestAbruptProcessExitReleasesLease(t *testing.T) {
	if dir := os.Getenv("SMALLWINNER_TEST_CHILD"); dir != "" {
		s, err := OpenStore(dir)
		if err != nil {
			os.Exit(2)
		}
		cfg := s.Snapshot()
		cfg.ThermalLimit = 82
		if _, err = s.Update(cfg); err != nil {
			os.Exit(3)
		}
		if err = os.WriteFile(filepath.Join(dir, "ready"), []byte("ready"), 0600); err != nil {
			os.Exit(4)
		}
		time.Sleep(time.Hour)
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestAbruptProcessExitReleasesLease$")
	cmd.Env = append(os.Environ(), "SMALLWINNER_TEST_CHILD="+dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child startup timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Snapshot().ThermalLimit != 82 {
		t.Fatal("committed config lost")
	}
}
func TestConcurrentControllerUpdates(t *testing.T) {
	c, _ := testController(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := c.Observe(context.Background(), Observation{Catalog[i%len(Catalog)].Package, float64(35 + i*2)})
			if err != nil {
				t.Error(err)
			}
			_ = c.Snapshot()
			_ = c.Config()
		}(i)
	}
	wg.Wait()
	st := c.Snapshot()
	if st.Device.Resources != st.Desired {
		t.Fatal("state does not match applied resources")
	}
}
