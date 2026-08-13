package shell

import (
	"bytes"
	"context"
	"os/exec"
)

type Runner struct {
	Sudo bool
}

func New(sudo bool) *Runner {
	return &Runner{Sudo: sudo}
}

func (r *Runner) Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	if r.Sudo {
		args = append([]string{name}, args...)
		return exec.CommandContext(ctx, "sudo", args...)
	}
	return exec.CommandContext(ctx, name, args...)
}

func (r *Runner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := r.Command(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func (r *Runner) Start(ctx context.Context, name string, args ...string) error {
	return r.Command(ctx, name, args...).Start()
}
