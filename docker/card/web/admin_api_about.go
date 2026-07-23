package web

import (
	"card/build"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiAbout(w http.ResponseWriter, r *http.Request) {
	// served from a TTL cache so the About page can poll frequently (to surface
	// updates without a manual refresh) without hitting Docker Hub each time
	latestVersion := latestVersionCache.get()
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

func (app *App) adminApiReleases(w http.ResponseWriter, r *http.Request) {
	type release struct {
		Version   string `json:"version"`
		Name      string `json:"name"`
		Body      string `json:"body"`
		Date      string `json:"date"`
		URL       string `json:"url"`
		IsCurrent bool   `json:"isCurrent"`
	}

	empty := map[string]interface{}{"releases": []release{}}

	running := build.Version
	latest := strings.TrimSpace(r.URL.Query().Get("latest"))

	client := &http.Client{Timeout: 10e9}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/boltcard/hub/releases?per_page=100", nil)
	if err != nil {
		writeJSON(w, empty)
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, empty)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		writeJSON(w, empty)
		return
	}

	var ghReleases []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		HTMLURL     string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghReleases); err != nil {
		writeJSON(w, empty)
		return
	}

	byVersion := map[string]release{}
	var versions []string
	for _, g := range ghReleases {
		ver := strings.TrimPrefix(g.TagName, "v")
		if !isVersion(ver) {
			continue
		}
		if _, dup := byVersion[ver]; dup {
			continue
		}
		name := g.Name
		if name == "" {
			name = "v" + ver
		}
		byVersion[ver] = release{
			Version:   ver,
			Name:      name,
			Body:      g.Body,
			Date:      g.PublishedAt,
			URL:       g.HTMLURL,
			IsCurrent: ver == running,
		}
		versions = append(versions, ver)
	}

	selected := selectReleases(running, latest, versions)
	releases := make([]release, 0, len(selected))
	for _, v := range selected {
		releases = append(releases, byVersion[v])
	}

	writeJSON(w, map[string]interface{}{"releases": releases})
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
