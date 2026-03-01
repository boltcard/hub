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

// CheckLatestVersion fetches the build.go file from GitHub and parses the version string.
func CheckLatestVersion() string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/boltcard/hub/main/docker/card/build/build.go")
	if err != nil {
		log.Warn("CheckLatestVersion: fetch failed: ", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Warn("CheckLatestVersion: unexpected status: ", resp.StatusCode)
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("CheckLatestVersion: read failed: ", err)
		return ""
	}

	re := regexp.MustCompile(`Version string = "(\d+\.\d+\.\d+)"`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		log.Warn("CheckLatestVersion: version not found in build.go")
		return ""
	}

	return string(matches[1])
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

	log.Info("TriggerUpdate: project dir = ", projectDir)

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
		"Cmd": ["sh", "-c", "docker compose pull && docker compose up -d --no-build"],
		"WorkingDir": "/project",
		"HostConfig": {
			"AutoRemove": true,
			"Binds": [
				"/var/run/docker.sock:/var/run/docker.sock",
				"%s:/project"
			]
		}
	}`, projectDir)

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
