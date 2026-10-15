package system

import (
	"context"
	"os/exec"
	"time"
)

func Run(command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	return string(output), err
}
