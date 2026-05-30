//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gnodux/adb-link/tests/testutil"
)

func TestMain(m *testing.M) {
	if !testutil.PodmanAvailable() {
		fmt.Fprintln(os.Stderr, "podman not available, skipping integration tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
