package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// WaitForTCP polls until addr accepts a TCP connection or timeout expires.
func WaitForTCP(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for TCP %s after %v", addr, timeout)
}

// WaitForSQL polls until the database accepts connections and responds to Ping.
func WaitForSQL(t *testing.T, driver, dsn string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open(driver, dsn)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err = db.PingContext(ctx)
			cancel()
			db.Close()
			if err == nil {
				return
			}
			lastErr = err
		} else {
			lastErr = err
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("timeout waiting for SQL %s after %v: %v", driver, timeout, lastErr)
}

// WaitForHTTP polls until the URL returns a non-error HTTP response.
func WaitForHTTP(t *testing.T, rawURL string, timeout time.Duration) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(rawURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return
			}
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("timeout waiting for HTTP %s after %v: %v", rawURL, timeout, lastErr)
}
