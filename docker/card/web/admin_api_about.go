package web

import (
	"card/build"
	"card/phoenix"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiAbout(w http.ResponseWriter, r *http.Request) {
	latestVersion := CheckLatestVersion()
	updateAvailable := false
	if latestVersion != "" {
		updateAvailable = CompareVersions(build.Version, latestVersion) == 1
	}

	phoenixdVersion := ""
	info, err := phoenix.GetNodeInfo()
	if err == nil {
		phoenixdVersion, _, _ = strings.Cut(info.Version, "-")
	} else {
		log.Warn("phoenix info error: ", err)
	}

	writeJSON(w, map[string]interface{}{
		"version":         build.Version,
		"buildDate":       build.Date,
		"buildTime":       build.Time,
		"phoenixdVersion": phoenixdVersion,
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

	if commits == nil {
		commits = []commit{}
	}

	writeJSON(w, map[string]interface{}{"commits": commits})
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
