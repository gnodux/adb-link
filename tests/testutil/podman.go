package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// Container represents a running podman container.
type Container struct {
	Name     string
	Image    string
	ID       string
	HostPort string
}

// PodmanAvailable returns true if podman is installed and functional.
func PodmanAvailable() bool {
	cmd := exec.Command("podman", "info")
	return cmd.Run() == nil
}

// StartContainer launches a container with the given image, env vars, and optional command.
// It maps a random host port to the given containerPort.
// Registers t.Cleanup to stop and remove the container.
func StartContainer(t *testing.T, image string, containerPort int, env []string, cmd []string) *Container {
	t.Helper()

	hostPort := FreePort(t)
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	name := fmt.Sprintf("adb-link-test-%s-%s", strings.ReplaceAll(t.Name(), "/", "-"), hex.EncodeToString(randBytes))

	args := []string{"run", "-d", "--rm", "--replace", "--name", name}
	args = append(args, "-p", fmt.Sprintf("%d:%d", hostPort, containerPort))
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, image)
	args = append(args, cmd...)

	out, err := exec.Command("podman", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("podman run failed: %v\n%s", err, string(out))
	}

	id := strings.TrimSpace(string(out))
	c := &Container{
		Name:     name,
		Image:    image,
		ID:       id,
		HostPort: fmt.Sprintf("%d", hostPort),
	}

	t.Cleanup(func() {
		StopContainer(c)
	})

	return c
}

// StopContainer stops and removes a container.
func StopContainer(c *Container) {
	exec.Command("podman", "stop", "-t", "5", c.ID).Run()
	exec.Command("podman", "rm", "-f", c.ID).Run()
}

// ContainerLogs returns the stdout/stderr logs of a container.
func ContainerLogs(c *Container) string {
	out, _ := exec.Command("podman", "logs", c.ID).CombinedOutput()
	return string(out)
}
