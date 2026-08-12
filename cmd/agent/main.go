package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type VersionManifest struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	SHA256      string `json:"sha256"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

const (
	defaultCheckInterval = 15 * time.Second
	workDir              = "/data/local/tmp"
	backupDir            = "/data/local/tmp/backup_previous"
	versionFile          = "/data/local/tmp/current_version.txt"
	githubRepo           = "abhishek-8285/avandab"
	healthCheckTimeout   = 10 * time.Second
)

func main() {
	manifestURL := os.Getenv("UPDATE_MANIFEST_URL")
	if manifestURL == "" {
		manifestURL = fmt.Sprintf("https://github.com/%s/releases/latest/download/manifest.json", githubRepo)
	}

	intervalStr := os.Getenv("CHECK_INTERVAL")
	checkInterval := defaultCheckInterval
	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil {
			checkInterval = d
		}
	}

	log.Printf("Starting Tecno Pova 2 Resilience Agent...")
	log.Printf("Target GitHub Repo: %s", githubRepo)
	log.Printf("Manifest URL: %s", manifestURL)
	log.Printf("Polling Interval: %s", checkInterval)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Initial check on boot
	checkAndUpdate(manifestURL)

	for range ticker.C {
		checkAndUpdate(manifestURL)
	}
}

func checkAndUpdate(manifestURL string) {
	manifest, err := fetchManifest(manifestURL)
	if err != nil {
		log.Printf("[Agent Error] Failed to fetch manifest from %s: %v", manifestURL, err)
		verifyAndRecoverRunningServer()
		return
	}

	currentVersion := getCurrentVersion()
	if manifest.Version == currentVersion {
		verifyAndRecoverRunningServer()
		return
	}

	log.Printf("[Agent Update] New version detected: %s (Current: %s)", manifest.Version, currentVersion)
	if err := applyUpdateWithRollback(manifest); err != nil {
		log.Printf("[Agent Error] Update failed: %v. Initiating automatic rollback...", err)
		if rollbackErr := performRollback(); rollbackErr != nil {
			log.Printf("[Agent CRITICAL] Rollback failed: %v", rollbackErr)
		} else {
			log.Printf("[Agent SUCCESS] Successfully rolled back to previous working version")
		}
		return
	}

	log.Printf("[Agent Success] Successfully updated Tecno Pova 2 to version %s", manifest.Version)
}

func fetchManifest(url string) (*VersionManifest, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TecnoPova2-AutoUpdateAgent")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var m VersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func getCurrentVersion() string {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func applyUpdateWithRollback(m *VersionManifest) error {
	tmpTar := filepath.Join(workDir, "update.tar.gz")
	defer func() { _ = os.Remove(tmpTar) }()

	log.Printf("[Agent] Downloading payload from %s...", m.DownloadURL)
	if err := downloadFile(tmpTar, m.DownloadURL); err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	if m.SHA256 != "" {
		hash, err := fileSHA256(tmpTar)
		if err != nil {
			return fmt.Errorf("hash calculation error: %w", err)
		}
		if !strings.EqualFold(hash, m.SHA256) {
			return fmt.Errorf("hash mismatch: expected %s, got %s", m.SHA256, hash)
		}
		log.Printf("[Agent] SHA256 verified successfully")
	}

	// Step 1: Backup current working binary and assets before replacing
	log.Printf("[Agent] Backing up current deployment before upgrade...")
	_ = os.RemoveAll(backupDir)
	_ = os.MkdirAll(backupDir, 0755)
	_ = copyFile(filepath.Join(workDir, "server"), filepath.Join(backupDir, "server"))
	_ = copyFile(versionFile, filepath.Join(backupDir, "current_version.txt"))

	// Step 2: Stop old process cleanly
	log.Printf("[Agent] Terminating existing server process...")
	_ = exec.Command("pkill", "-9", "-f", "server").Run()
	_ = exec.Command("su", "-c", "pkill -9 -f server").Run()
	time.Sleep(1500 * time.Millisecond)
	_ = os.Remove(filepath.Join(workDir, "server"))

	// Step 3: Extract new update
	log.Printf("[Agent] Extracting updated files...")
	if err := untarGz(tmpTar, workDir); err != nil {
		return fmt.Errorf("unpack error: %w", err)
	}
	_ = os.Chmod(filepath.Join(workDir, "server"), 0755)

	// Step 4: Restart Server
	log.Printf("[Agent] Launching new server build...")
	restartServer()

	// Step 5: Perform Health Check (Verify new binary isn't crashing)
	log.Printf("[Agent] Verifying server health post-deployment...")
	if !checkServerHealth() {
		return fmt.Errorf("health check failed: server crashed or failed to respond on port 8092")
	}

	log.Printf("[Agent] Health check PASSED.")
	return os.WriteFile(versionFile, []byte(m.Version), 0644)
}

func performRollback() error {
	backupServer := filepath.Join(backupDir, "server")
	if _, err := os.Stat(backupServer); err != nil {
		return fmt.Errorf("no backup binary found to restore")
	}

	_ = exec.Command("pkill", "-9", "server").Run()
	time.Sleep(1 * time.Second)

	if err := copyFile(backupServer, filepath.Join(workDir, "server")); err != nil {
		return fmt.Errorf("failed to restore backup binary: %w", err)
	}
	_ = os.Chmod(filepath.Join(workDir, "server"), 0755)

	backupVer := filepath.Join(backupDir, "current_version.txt")
	if data, err := os.ReadFile(backupVer); err == nil {
		_ = os.WriteFile(versionFile, data, 0644)
	}

	restartServer()
	log.Printf("[Agent] Restored previous server binary and restarted.")
	return nil
}

func verifyAndRecoverRunningServer() {
	// Always enforce Airplane Mode ON and Wi-Fi ON to prevent MediaTek empty SIM slot crash loop
	_ = exec.Command("su", "-c", "settings put global airplane_mode_on 1; am broadcast -a android.intent.action.AIRPLANE_MODE --ez state true; svc wifi enable").Run()

	if !checkServerHealth() {
		log.Printf("[Agent Recover] Warning: Server is not responding on port 8092. Attempting auto-restart...")
		_ = exec.Command("pkill", "-9", "server").Run()
		time.Sleep(1 * time.Second)
		restartServer()
	}
}

func checkServerHealth() bool {
	deadline := time.Now().Add(healthCheckTimeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get("http://127.0.0.1:8092/")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMethodNotAllowed {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func restartServer() {
	startScript := filepath.Join(workDir, "start.sh")
	if _, err := os.Stat(startScript); err == nil {
		cmd := exec.Command("/system/bin/sh", startScript)
		cmd.Dir = workDir
		if err := cmd.Start(); err != nil {
			launchServerDirectly()
		}
	} else {
		launchServerDirectly()
	}
}

func launchServerDirectly() {
	cmd := exec.Command("./server")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"PORT=8092",
		"ENV=production",
		"DATABASE_URL=file:mvtms.db?_journal_mode=WAL&_synchronous=OFF&cache=shared&mode=rwc",
	)
	_ = cmd.Start()
}

func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "TecnoPova2-AutoUpdateAgent")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Timeout: 60 * time.Second, Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}

func untarGz(src string, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return err
			}
			_ = outFile.Close()
		}
	}
	return nil
}
