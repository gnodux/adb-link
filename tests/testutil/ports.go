package testutil

import (
	"net"
	"testing"
)

// FreePort returns a free TCP port by binding to :0 and releasing.
func FreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
