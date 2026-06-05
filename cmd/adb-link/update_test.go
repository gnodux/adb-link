package main

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	os, arch := detectPlatform()
	if os != runtime.GOOS {
		t.Errorf("detectPlatform() os = %q, want %q", os, runtime.GOOS)
	}
	if arch != runtime.GOARCH {
		t.Errorf("detectPlatform() arch = %q, want %q", arch, runtime.GOARCH)
	}
}

func TestSelectAsset_LinuxAmd64(t *testing.T) {
	assets := []githubAsset{
		{Name: "adb-link-v1.0.9-darwin-amd64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-amd64"},
		{Name: "adb-link-v1.0.9-darwin-arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-arm64"},
		{Name: "adb-link-v1.0.9-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64"},
		{Name: "adb-link-v1.0.9-linux-arm64.tar.gz", BrowserDownloadURL: "https://example.com/linux-arm64"},
		{Name: "adb-link-v1.0.9-windows-amd64.zip", BrowserDownloadURL: "https://example.com/windows-amd64"},
		{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums"},
	}

	url, name, err := selectAsset(assets, "linux", "amd64")
	if err != nil {
		t.Fatalf("selectAsset() error = %v", err)
	}
	if url != "https://example.com/linux-amd64" {
		t.Errorf("selectAsset() url = %q, want %q", url, "https://example.com/linux-amd64")
	}
	if name != "adb-link-v1.0.9-linux-amd64.tar.gz" {
		t.Errorf("selectAsset() name = %q, want %q", name, "adb-link-v1.0.9-linux-amd64.tar.gz")
	}
}

func TestSelectAsset_DarwinArm64(t *testing.T) {
	assets := []githubAsset{
		{Name: "adb-link-v1.0.9-darwin-amd64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-amd64"},
		{Name: "adb-link-v1.0.9-darwin-arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-arm64"},
		{Name: "adb-link-v1.0.9-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64"},
	}

	url, _, err := selectAsset(assets, "darwin", "arm64")
	if err != nil {
		t.Fatalf("selectAsset() error = %v", err)
	}
	if url != "https://example.com/darwin-arm64" {
		t.Errorf("selectAsset() url = %q, want %q", url, "https://example.com/darwin-arm64")
	}
}

func TestSelectAsset_WindowsAmd64(t *testing.T) {
	assets := []githubAsset{
		{Name: "adb-link-v1.0.9-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64"},
		{Name: "adb-link-v1.0.9-windows-amd64.zip", BrowserDownloadURL: "https://example.com/windows-amd64"},
	}

	url, name, err := selectAsset(assets, "windows", "amd64")
	if err != nil {
		t.Fatalf("selectAsset() error = %v", err)
	}
	if url != "https://example.com/windows-amd64" {
		t.Errorf("selectAsset() url = %q, want %q", url, "https://example.com/windows-amd64")
	}
	if name != "adb-link-v1.0.9-windows-amd64.zip" {
		t.Errorf("selectAsset() name = %q, want %q", name, "adb-link-v1.0.9-windows-amd64.zip")
	}
}

func TestSelectAsset_UnsupportedPlatform(t *testing.T) {
	assets := []githubAsset{
		{Name: "adb-link-v1.0.9-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64"},
		{Name: "adb-link-v1.0.9-darwin-arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-arm64"},
	}

	_, _, err := selectAsset(assets, "freebsd", "amd64")
	if err == nil {
		t.Fatal("selectAsset() expected error for unsupported platform, got nil")
	}
}

func TestSelectAsset_NoAssets(t *testing.T) {
	_, _, err := selectAsset(nil, "linux", "amd64")
	if err == nil {
		t.Fatal("selectAsset() expected error for empty assets, got nil")
	}
}

func TestSelectAsset_SkipsNonArchive(t *testing.T) {
	assets := []githubAsset{
		{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums"},
		{Name: "adb-link-v1.0.9-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64"},
	}

	url, _, err := selectAsset(assets, "linux", "amd64")
	if err != nil {
		t.Fatalf("selectAsset() error = %v", err)
	}
	if url != "https://example.com/linux-amd64" {
		t.Errorf("selectAsset() url = %q, want %q", url, "https://example.com/linux-amd64")
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		patterns []string
		want     bool
	}{
		{"match first", "adb-link-linux-amd64.tar.gz", []string{"linux"}, true},
		{"match second", "adb-link-darwin-arm64.tar.gz", []string{"darwin", "macos"}, true},
		{"no match", "adb-link-linux-amd64.tar.gz", []string{"darwin", "windows"}, false},
		{"empty patterns", "adb-link-linux-amd64.tar.gz", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAny(tt.input, tt.patterns)
			if got != tt.want {
				t.Errorf("matchesAny(%q, %v) = %v, want %v", tt.input, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestPlatformOSPatterns(t *testing.T) {
	tests := []struct {
		os   string
		want []string
	}{
		{"darwin", []string{"darwin", "macos", "mac", "osx"}},
		{"linux", []string{"linux"}},
		{"windows", []string{"windows", "win64", "win32", "win"}},
		{"freebsd", []string{"freebsd"}},
	}
	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			got := platformOSPatterns(tt.os)
			if len(got) != len(tt.want) {
				t.Errorf("platformOSPatterns(%q) = %v, want %v", tt.os, got, tt.want)
			}
		})
	}
}

func TestPlatformArchPatterns(t *testing.T) {
	tests := []struct {
		arch string
		want []string
	}{
		{"amd64", []string{"amd64", "x86_64", "x64"}},
		{"arm64", []string{"arm64", "aarch64"}},
		{"riscv64", []string{"riscv64"}},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			got := platformArchPatterns(tt.arch)
			if len(got) != len(tt.want) {
				t.Errorf("platformArchPatterns(%q) = %v, want %v", tt.arch, got, tt.want)
			}
		})
	}
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		equal   bool
	}{
		{"same version", "1.0.8", "1.0.8", true},
		{"newer available", "1.0.8", "1.0.9", false},
		{"major bump", "1.0.8", "2.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.current == tt.latest
			if got != tt.equal {
				t.Errorf("version comparison %q == %q = %v, want %v", tt.current, tt.latest, got, tt.equal)
			}
		})
	}
}
