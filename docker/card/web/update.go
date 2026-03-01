package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const dockerHubImage = "boltcard/card"

// CheckLatestVersion queries Docker Hub for the version label on the latest image.
func CheckLatestVersion() string {
	client := &http.Client{Timeout: 15 * time.Second}

	// 1. Get anonymous token for the public repo
	tokenURL := fmt.Sprintf(
		"https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull",
		dockerHubImage,
	)
	resp, err := client.Get(tokenURL)
	if err != nil {
		log.Warn("CheckLatestVersion: token request failed: ", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Warn("CheckLatestVersion: token status: ", resp.StatusCode)
		return ""
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Warn("CheckLatestVersion: token decode failed: ", err)
		return ""
	}

	// 2. Fetch manifest to get config digest
	manifestURL := fmt.Sprintf(
		"https://registry-1.docker.io/v2/%s/manifests/latest",
		dockerHubImage,
	)
	req, _ := http.NewRequest("GET", manifestURL, nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.Token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp2, err := client.Do(req)
	if err != nil {
		log.Warn("CheckLatestVersion: manifest fetch failed: ", err)
		return ""
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		log.Warn("CheckLatestVersion: manifest status: ", resp2.StatusCode)
		return ""
	}

	var manifest struct {
		Config struct {
			Digest string `json:"digest"`
		} `json:"config"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&manifest); err != nil {
		log.Warn("CheckLatestVersion: manifest decode failed: ", err)
		return ""
	}

	if manifest.Config.Digest == "" {
		log.Warn("CheckLatestVersion: no config digest in manifest")
		return ""
	}

	// 3. Fetch config blob to read version label
	blobURL := fmt.Sprintf(
		"https://registry-1.docker.io/v2/%s/blobs/%s",
		dockerHubImage, manifest.Config.Digest,
	)
	req2, _ := http.NewRequest("GET", blobURL, nil)
	req2.Header.Set("Authorization", "Bearer "+tokenResp.Token)

	resp3, err := client.Do(req2)
	if err != nil {
		log.Warn("CheckLatestVersion: blob fetch failed: ", err)
		return ""
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != 200 {
		log.Warn("CheckLatestVersion: blob status: ", resp3.StatusCode)
		return ""
	}

	var config struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"config"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&config); err != nil {
		log.Warn("CheckLatestVersion: blob decode failed: ", err)
		return ""
	}

	version := config.Config.Labels["org.opencontainers.image.version"]
	if version == "" {
		log.Warn("CheckLatestVersion: no version label on image")
		return ""
	}

	// Validate format
	if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, version); !matched {
		log.Warn("CheckLatestVersion: invalid version format: ", version)
		return ""
	}

	return version
}

// CompareVersions returns 1 if latest > current, 0 if equal, -1 if latest < current.
func CompareVersions(current, latest string) int {
	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	for i := range 3 {
		c, _ := strconv.Atoi(safeIndex(currentParts, i))
		l, _ := strconv.Atoi(safeIndex(latestParts, i))
		if l > c {
			return 1
		}
		if l < c {
			return -1
		}
	}
	return 0
}

func safeIndex(parts []string, i int) string {
	if i < len(parts) {
		return parts[i]
	}
	return "0"
}

// dockerTransport returns an http.Client that talks to the Docker daemon over Unix socket.
func dockerTransport() *http.Client {
	return &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
}

// dockerGet performs a GET request to the Docker API.
func dockerGet(path string) (*http.Response, error) {
	client := dockerTransport()
	return client.Get("http://localhost" + path)
}

// dockerPost performs a POST request to the Docker API.
func dockerPost(path string, contentType string, body io.Reader) (*http.Response, error) {
	client := dockerTransport()
	return client.Post("http://localhost"+path, contentType, body)
}

// TriggerUpdate creates a disposable updater container that pulls new images and recreates containers.
func TriggerUpdate() error {
	// 1. Inspect own container to get compose project working directory
	resp, err := dockerGet("/containers/card/json")
	if err != nil {
		return fmt.Errorf("inspect card container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("inspect card container: status %d: %s", resp.StatusCode, body)
	}

	var inspectResult struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&inspectResult); err != nil {
		return fmt.Errorf("decode inspect result: %w", err)
	}

	projectDir := inspectResult.Config.Labels["com.docker.compose.project.working_dir"]
	if projectDir == "" {
		return fmt.Errorf("compose project working dir label not found")
	}

	projectName := inspectResult.Config.Labels["com.docker.compose.project"]
	if projectName == "" {
		return fmt.Errorf("compose project name label not found")
	}

	log.Info("TriggerUpdate: project dir = ", projectDir, ", name = ", projectName)

	// 2. Check if updater already exists
	resp2, err := dockerGet("/containers/hub-updater/json")
	if err == nil {
		resp2.Body.Close()
		if resp2.StatusCode == 200 {
			return fmt.Errorf("update already in progress")
		}
	}

	// 3. Pull docker:cli image
	log.Info("TriggerUpdate: pulling docker:cli image")
	resp3, err := dockerPost("/images/create?fromImage=docker&tag=cli", "application/json", nil)
	if err != nil {
		return fmt.Errorf("pull docker:cli: %w", err)
	}
	defer resp3.Body.Close()
	// Read the full response to wait for pull to complete
	io.Copy(io.Discard, resp3.Body)

	if resp3.StatusCode != 200 {
		return fmt.Errorf("pull docker:cli: status %d", resp3.StatusCode)
	}

	// 4. Create updater container
	log.Info("TriggerUpdate: creating updater container")
	createBody := fmt.Sprintf(`{
		"Image": "docker:cli",
		"Cmd": ["sh", "-c", "docker compose -p %s pull card && docker compose -p %s up -d --no-build card"],
		"WorkingDir": "%s",
		"HostConfig": {
			"AutoRemove": true,
			"Binds": [
				"/var/run/docker.sock:/var/run/docker.sock",
				"%s:%s"
			]
		}
	}`, projectName, projectName, projectDir, projectDir, projectDir)

	resp4, err := dockerPost(
		"/containers/create?name=hub-updater",
		"application/json",
		strings.NewReader(createBody),
	)
	if err != nil {
		return fmt.Errorf("create updater container: %w", err)
	}
	defer resp4.Body.Close()

	if resp4.StatusCode != 201 {
		body, _ := io.ReadAll(resp4.Body)
		return fmt.Errorf("create updater container: status %d: %s", resp4.StatusCode, body)
	}

	var createResult struct {
		Id string `json:"Id"`
	}
	if err := json.NewDecoder(resp4.Body).Decode(&createResult); err != nil {
		return fmt.Errorf("decode create result: %w", err)
	}

	// 5. Start updater container
	log.Info("TriggerUpdate: starting updater container ", createResult.Id[:12])
	resp5, err := dockerPost("/containers/"+createResult.Id+"/start", "application/json", nil)
	if err != nil {
		return fmt.Errorf("start updater container: %w", err)
	}
	defer resp5.Body.Close()

	if resp5.StatusCode != 204 {
		body, _ := io.ReadAll(resp5.Body)
		return fmt.Errorf("start updater container: status %d: %s", resp5.StatusCode, body)
	}

	log.Info("TriggerUpdate: updater container started successfully")
	return nil
}
