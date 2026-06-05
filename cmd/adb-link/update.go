package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const githubRepo = "gnodux/adb-link"

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func runUpdate(currentVersion string) {
	platOS, platArch := detectPlatform()

	fmt.Printf("adb-link %s (%s/%s)\n", currentVersion, platOS, platArch)
	fmt.Println("Checking for updates...")

	rel, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	latestVersion := strings.TrimPrefix(rel.TagName, "v")
	if latestVersion == currentVersion {
		fmt.Printf("Already up to date (v%s).\n", currentVersion)
		return
	}

	fmt.Printf("New version available: v%s → v%s\n", currentVersion, latestVersion)

	assetURL, assetName, err := selectAsset(rel.Assets, platOS, platArch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Downloading %s...\n", assetName)

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine executable path: %v\n", err)
		os.Exit(1)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot resolve executable path: %v\n", err)
		os.Exit(1)
	}

	if err := downloadAndReplace(assetURL, assetName, exePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to v%s successfully.\n", latestVersion)
}

func detectPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &rel, nil
}

func selectAsset(assets []githubAsset, osName, arch string) (url, name string, err error) {
	osPatterns := platformOSPatterns(osName)
	archPatterns := platformArchPatterns(arch)

	archiveSuffix := ".tar.gz"
	if osName == "windows" {
		archiveSuffix = ".zip"
	}

	for _, a := range assets {
		if !strings.HasSuffix(a.Name, archiveSuffix) {
			continue
		}
		lower := strings.ToLower(a.Name)
		if matchesAny(lower, osPatterns) && matchesAny(lower, archPatterns) {
			return a.BrowserDownloadURL, a.Name, nil
		}
	}

	var available []string
	for _, a := range assets {
		available = append(available, a.Name)
	}
	return "", "", fmt.Errorf("no asset found for %s/%s.\nAvailable assets:\n  %s",
		osName, arch, strings.Join(available, "\n  "))
}

func platformOSPatterns(osName string) []string {
	switch osName {
	case "darwin":
		return []string{"darwin", "macos", "mac", "osx"}
	case "linux":
		return []string{"linux"}
	case "windows":
		return []string{"windows", "win64", "win32", "win"}
	default:
		return []string{osName}
	}
}

func platformArchPatterns(arch string) []string {
	switch arch {
	case "amd64":
		return []string{"amd64", "x86_64", "x64"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	default:
		return []string{arch}
	}
}

func matchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(name, p) {
			return true
		}
	}
	return false
}

func downloadAndReplace(assetURL, assetName, exePath string) error {
	tmpFile, err := os.CreateTemp("", "adb-link-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	resp, err := http.Get(assetURL)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("save download: %w", err)
	}
	tmpFile.Close()

	newBinary, err := extractBinary(tmpPath, assetName)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	defer os.Remove(newBinary)

	backupPath := exePath + ".bak"
	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("backup existing binary: %w", err)
	}

	if err := copyFile(newBinary, exePath); err != nil {
		os.Rename(backupPath, exePath)
		return fmt.Errorf("install new binary: %w", err)
	}

	if err := os.Chmod(exePath, 0755); err != nil {
		os.Rename(backupPath, exePath)
		return fmt.Errorf("set permissions: %w", err)
	}

	os.Remove(backupPath)
	return nil
}

func extractBinary(archivePath, assetName string) (string, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	binName := "adb-link"
	if runtime.GOOS == "windows" {
		binName = "adb-link.exe"
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read tar: %w", err)
		}
		base := filepath.Base(hdr.Name)
		if base == binName && hdr.Typeflag == tar.TypeReg {
			tmpBin, err := os.CreateTemp("", "adb-link-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmpBin, tr); err != nil {
				tmpBin.Close()
				os.Remove(tmpBin.Name())
				return "", err
			}
			tmpBin.Close()
			return tmpBin.Name(), nil
		}
	}
	return "", fmt.Errorf("binary '%s' not found in archive", binName)
}

func extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	binName := "adb-link.exe"

	for _, f := range r.File {
		if filepath.Base(f.Name) == binName && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			tmpBin, err := os.CreateTemp("", "adb-link-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmpBin, rc); err != nil {
				tmpBin.Close()
				os.Remove(tmpBin.Name())
				return "", err
			}
			tmpBin.Close()
			return tmpBin.Name(), nil
		}
	}
	return "", fmt.Errorf("binary '%s' not found in archive", binName)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
