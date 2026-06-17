package web

import (
	"card/build"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiAbout(w http.ResponseWriter, r *http.Request) {
	latestVersion := CheckLatestVersion()
	updateAvailable := false
	if latestVersion != "" {
		updateAvailable = CompareVersions(build.Version, latestVersion) == 1
	}

	writeJSON(w, map[string]interface{}{
		"version":         build.Version,
		"buildDate":       build.Date,
		"buildTime":       build.Time,
		"latestVersion":   latestVersion,
		"updateAvailable": updateAvailable,
	})
}

func (app *App) adminApiTriggerUpdate(w http.ResponseWriter, r *http.Request) {
	err := TriggerUpdate()
	if err != nil {
		log.Error("update trigger error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (app *App) adminApiLogs(w http.ResponseWriter, r *http.Request) {
	resp, err := dockerGet("/containers/card/logs?stdout=1&stderr=1&tail=20&timestamps=0")
	if err != nil {
		log.Warn("adminApiLogs: docker logs error: ", err)
		writeJSON(w, map[string]interface{}{"logs": []string{}})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSON(w, map[string]interface{}{"logs": []string{}})
		return
	}

	// Docker multiplexed stream: each frame has 8-byte header
	var lines []string
	raw := body
	for len(raw) >= 8 {
		size := int(raw[4])<<24 | int(raw[5])<<16 | int(raw[6])<<8 | int(raw[7])
		raw = raw[8:]
		if size > len(raw) {
			size = len(raw)
		}
		line := strings.TrimRight(string(raw[:size]), "\n")
		line = ansiToHTML(line)
		if line != "" {
			lines = append(lines, line)
		}
		raw = raw[size:]
	}

	writeJSON(w, map[string]interface{}{"logs": lines})
}

func (app *App) adminApiCommits(w http.ResponseWriter, r *http.Request) {
	type commit struct {
		Sha     string `json:"sha"`
		Message string `json:"message"`
		Date    string `json:"date"`
		Version string `json:"version"`
	}

	var commits []commit

	client := &http.Client{Timeout: 10e9}
	url := "https://api.github.com/repos/boltcard/hub/commits?per_page=10"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		writeJSON(w, map[string]interface{}{"commits": []commit{}})
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, map[string]interface{}{"commits": []commit{}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var ghCommits []struct {
			Sha    string `json:"sha"`
			Commit struct {
				Message string `json:"message"`
				Author  struct {
					Date string `json:"date"`
				} `json:"author"`
			} `json:"commit"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ghCommits); err == nil {
			for _, c := range ghCommits {
				msg := c.Commit.Message
				if idx := strings.Index(msg, "\n"); idx != -1 {
					msg = msg[:idx]
				}
				commits = append(commits, commit{
					Sha:     c.Sha,
					Message: msg,
					Date:    c.Commit.Author.Date,
				})
			}
		}
	}

	// Annotate each commit with the app version recorded in build/build.go at
	// that commit. The commits-list API doesn't return file contents, so this
	// needs one extra fetch per commit; results are cached by SHA (immutable)
	// and fetched concurrently so a cold load doesn't serialise 10 round-trips.
	var wg sync.WaitGroup
	for i := range commits {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			commits[i].Version = fetchCommitVersion(client, commits[i].Sha)
		}(i)
	}
	wg.Wait()

	if commits == nil {
		commits = []commit{}
	}

	writeJSON(w, map[string]interface{}{"commits": commits})
}

var buildVersionRegex = regexp.MustCompile(`Version string = "([^"]+)"`)

// parseVersionFromBuildGo extracts the version string from build/build.go
// source, returning "" if no version declaration is found.
func parseVersionFromBuildGo(content string) string {
	m := buildVersionRegex.FindStringSubmatch(content)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

var (
	commitVersionMu    sync.Mutex
	commitVersionCache = map[string]string{}
)

// fetchCommitVersion returns the app version recorded in build/build.go at the
// given commit SHA, fetching it from the GitHub contents API. Results are
// memoised per SHA — a commit's tree is immutable, so the cache never goes
// stale. Returns "" on any error (rate limit, missing file in old commits).
func fetchCommitVersion(client *http.Client, sha string) string {
	if sha == "" {
		return ""
	}

	commitVersionMu.Lock()
	if v, ok := commitVersionCache[sha]; ok {
		commitVersionMu.Unlock()
		return v
	}
	commitVersionMu.Unlock()

	url := "https://api.github.com/repos/boltcard/hub/contents/docker/card/build/build.go?ref=" + sha
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	version := ""
	if resp.StatusCode == 200 {
		var payload struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil && payload.Encoding == "base64" {
			// GitHub wraps base64 content at 60 chars with newlines.
			decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(payload.Content, "\n", ""))
			if err == nil {
				version = parseVersionFromBuildGo(string(decoded))
			}
		}
	}

	commitVersionMu.Lock()
	commitVersionCache[sha] = version
	commitVersionMu.Unlock()
	return version
}

var versionRegex = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// isVersion reports whether s is a bare three-part numeric version (X.Y.Z).
func isVersion(s string) bool {
	return versionRegex.MatchString(s)
}

// selectReleases returns the release versions to display, newest-first, given
// the running version, the latest available version (may be "" or invalid),
// and the available release versions (each a bare "X.Y.Z"). The shown range is
// running <= ver <= upper, where upper is the latest version when it is a valid
// version greater than running, otherwise running. Non-version entries are
// skipped. Returns nil when running is not a valid version.
func selectReleases(running, latest string, versions []string) []string {
	if !isVersion(running) {
		return nil
	}
	upper := running
	if isVersion(latest) && CompareVersions(running, latest) == 1 {
		upper = latest
	}

	var out []string
	for _, v := range versions {
		if !isVersion(v) {
			continue
		}
		// running <= v <= upper
		if CompareVersions(running, v) >= 0 && CompareVersions(v, upper) >= 0 {
			out = append(out, v)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		// descending: out[i] before out[j] when out[i] > out[j]
		return CompareVersions(out[i], out[j]) == -1
	})
	return out
}

var ansiColorMap = map[string]string{
	"30": "#6b7280", // black/gray
	"31": "#ef4444", // red
	"32": "#22c55e", // green
	"33": "#eab308", // yellow
	"34": "#3b82f6", // blue
	"35": "#a855f7", // magenta
	"36": "#06b6d4", // cyan
	"37": "#d1d5db", // white
}

var ansiRegex = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// ansiToHTML converts ANSI color escape sequences to HTML span elements.
// Text between sequences is HTML-escaped for safe rendering.
func ansiToHTML(raw string) string {
	var b strings.Builder
	open := false
	last := 0

	for _, loc := range ansiRegex.FindAllStringSubmatchIndex(raw, -1) {
		// Write text before this match, HTML-escaped
		b.WriteString(html.EscapeString(raw[last:loc[0]]))

		param := raw[loc[2]:loc[3]]
		if param == "0" || param == "" {
			if open {
				b.WriteString("</span>")
				open = false
			}
		} else if color, ok := ansiColorMap[param]; ok {
			if open {
				b.WriteString("</span>")
			}
			b.WriteString(`<span style="color:` + color + `">`)
			open = true
		}
		last = loc[1]
	}

	b.WriteString(html.EscapeString(raw[last:]))
	if open {
		b.WriteString("</span>")
	}

	return b.String()
}
