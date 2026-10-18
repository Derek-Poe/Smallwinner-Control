package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Derek-Poe/Smallwinner-Control/internal/control"
	"github.com/Derek-Poe/Smallwinner-Control/internal/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	stateDir := flag.String("state", "state", "private local state directory (Linux/WSL)")
	port := flag.Int("port", 1717, "loopback port")
	demo := flag.Bool("demo", false, "print a deterministic policy walkthrough; no HTTP server")
	flag.Parse()
	store, err := control.OpenStore(*stateDir)
	if err != nil {
		return err
	}
	defer store.Close()
	controller := control.NewController(store, control.NewSimulator())
	if *demo {
		return walkthrough(controller)
	}
	token := os.Getenv("SMALLWINNER_TOKEN")
	if token == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		token = hex.EncodeToString(b)
	}
	handler, err := web.New(controller, token, *port)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 8192}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fmt.Printf("Smallwinner simulation: %s\nSession token (keep local): %s\n", handler.URL(), token)
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		_ = server.Close()
		return err
	}
	_, err = controller.Restore(context.Background())
	return err
}
func walkthrough(c *control.Controller) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	states := []control.State{c.Snapshot()}
	for _, o := range []control.Observation{{Foreground: "demo.game", Temperature: 35}, {Foreground: "demo.game", Temperature: 80}, {Foreground: "demo.game", Temperature: 72}, {Foreground: "demo.game", Temperature: 69}, {Foreground: "demo.home", Temperature: 35}} {
		st, err := c.Observe(context.Background(), o)
		if err != nil {
			return err
		}
		states = append(states, st)
	}
	return enc.Encode(states)
}
