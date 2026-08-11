// Example GitHub Releases Manifest Fetcher snippet for Tecno Pova 2 Agent
// The agent queries the GitHub REST API for the latest release manifest.json

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func fetchLatestGitHubManifest(repo string) (*VersionManifest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	var manifestURL string
	for _, asset := range release.Assets {
		if asset.Name == "manifest.json" {
			manifestURL = asset.BrowserDownloadURL
			break
		}
	}

	if manifestURL == "" {
		return nil, fmt.Errorf("manifest.json not found in latest release assets")
	}

	return fetchManifest(manifestURL)
}
