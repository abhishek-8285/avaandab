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
	versionFile          = "/data/local/tmp/current_version.txt"
	githubRepo           = "abhishek-8285/avaandab"
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

	log.Printf("Starting Tecno Pova 2 Remote Deployment Agent...")
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
		return
	}

	currentVersion := getCurrentVersion()
	if manifest.Version == currentVersion {
		// Version up to date
		return
	}

	log.Printf("[Agent Update] New version detected: %s (Current: %s)", manifest.Version, currentVersion)
	if err := applyUpdate(manifest); err != nil {
		log.Printf("[Agent Error] Failed to apply update %s: %v", manifest.Version, err)
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
	defer resp.Body.Close()

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

func applyUpdate(m *VersionManifest) error {
	tmpTar := filepath.Join(workDir, "update.tar.gz")
	defer os.Remove(tmpTar)

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

	log.Printf("[Agent] Terminating existing server process...")
	_ = exec.Command("pkill", "-9", "server").Run()

	log.Printf("[Agent] Extracting updated files...")
	if err := untarGz(tmpTar, workDir); err != nil {
		return fmt.Errorf("unpack error: %w", err)
	}

	_ = os.Chmod(filepath.Join(workDir, "server"), 0755)

	log.Printf("[Agent] Restarting application server...")
	startScript := filepath.Join(workDir, "start.sh")
	if _, err := os.Stat(startScript); err == nil {
		cmd := exec.Command("/system/bin/sh", startScript)
		cmd.Dir = workDir
		if err := cmd.Start(); err != nil {
			log.Printf("[Agent Warning] Failed to run start.sh: %v, launching server directly", err)
			launchServerDirectly()
		}
	} else {
		launchServerDirectly()
	}

	return os.WriteFile(versionFile, []byte(m.Version), 0644)
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
	defer out.Close()

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
	defer resp.Body.Close()

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
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func untarGz(src string, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

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
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}
