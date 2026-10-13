package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const webRoot = "/opt/cellapp/web"

type result struct {
	name string
	err  error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || n == 3000 {
		return errors.New("invalid PORT")
	}
	if _, err := os.Stat(filepath.Join(webRoot, "index.html")); err != nil {
		return errors.New("WEB_ROOT unavailable")
	}
	if info, err := os.Stat(filepath.Join(webRoot, "assets")); err != nil || !info.IsDir() {
		return errors.New("WEB_ROOT unavailable")
	}
	env := append(os.Environ(), "PORT="+port, "NODE_ENV=production", "WEB_ROOT="+webRoot, "LISTEN_ADDRESS=127.0.0.1:3000")
	server := exec.Command("/usr/local/bin/cellapp-server")
	caddy := exec.Command("/usr/bin/caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile")
	for _, cmd := range []*exec.Cmd{server, caddy} {
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := server.Start(); err != nil {
		return fmt.Errorf("server start failed: %w", err)
	}
	if err := caddy.Start(); err != nil {
		_ = server.Process.Signal(syscall.SIGTERM)
		_ = server.Wait()
		return fmt.Errorf("Caddy start failed: %w", err)
	}
	finished := make(chan result, 2)
	go func() { finished <- result{"server", server.Wait()} }()
	go func() { finished <- result{"Caddy", caddy.Wait()} }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	var first result
	stopping := false
	select {
	case first = <-finished:
	case <-signals:
		stopping = true
	}
	for _, cmd := range []*exec.Cmd{server, caddy} {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
	remaining := 2
	if !stopping {
		remaining = 1
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for remaining > 0 {
		select {
		case <-finished:
			remaining--
		case <-timer.C:
			_ = server.Process.Kill()
			_ = caddy.Process.Kill()
			for remaining > 0 {
				<-finished
				remaining--
			}
		}
	}
	if stopping {
		return nil
	}
	return fmt.Errorf("%s stopped unexpectedly: %v", first.name, first.err)
}
