# About page: Recent Releases Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the About page's raw "Recent Commits" list with "Recent Releases" — proper GitHub Releases (auto-published in CI on version bump from tidied commit subjects), scoped to the gap between the hub's running version and the latest available version.

**Architecture:** A CI `release` job tags `v{version}` and publishes a GitHub Release whenever `build/build.go`'s version changes. The card backend swaps its N+1 per-commit GitHub fetch for a single Releases-API call, filters the releases to the `[running, latest]` range with a pure testable helper, and the React About page renders them as release blocks.

**Tech Stack:** Go 1.25.11 (Gorilla Mux handlers, CGo/sqlite), React 19 + Vite + TypeScript + Tailwind v4 + shadcn/ui, GitHub Actions, `gh` CLI.

## Global Constraints

- Module name is `card`; imports are `card/build`, `card/web`, etc. Working dir for Go tests: `docker/card/`.
- Go tests need CGo (default-on) and run via `go test -race -count=1 ./...`.
- Admin API handler pattern: `func (app *App) adminApiX(w http.ResponseWriter, r *http.Request)`; routes registered in `web/admin_api.go` behind `adminApiAuth(...)`; JSON responses via `writeJSON(w, v)`.
- `CompareVersions(current, latest string) int` returns `1` when `latest > current`, `0` when equal, `-1` when `latest < current` (in `web/update.go`).
- Bump `Version` in `docker/card/build/build.go` for this PR: `0.21.0` → `0.22.0` (MINOR — new feature). Keep the `build/` line in `CLAUDE.md` ("currently \"0.21.0\"") in sync.
- Frontend: no new markdown dependency; React auto-escapes children — do not use `dangerouslySetInnerHTML` for release bodies.
- Node/npm in this dev env: `export NVM_DIR="/home/debian/.nvm" && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0` before npm; frontend build = `npm run build` in `docker/card/admin-ui`.
- Repo: `boltcard/hub`. Default branch `main`. Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

---

## File Structure

- `docker/card/web/admin_api_about.go` — Modify. Add `selectReleases` + `isVersion` + `versionRegex` (Task 1). Replace `adminApiCommits` with `adminApiReleases`; remove `fetchCommitVersion`, `commitVersionMu`, `commitVersionCache`, `parseVersionFromBuildGo`, `buildVersionRegex` and the `encoding/base64`/`sync` imports; add `sort` (Task 2). Keep `adminApiAbout`, `adminApiTriggerUpdate`, `adminApiLogs`, `ansiToHTML`, `ansiColorMap`, `ansiRegex`.
- `docker/card/web/admin_api_about_test.go` — Modify. Add `TestSelectReleases` (Task 1); remove `TestParseVersionFromBuildGo` (Task 2).
- `docker/card/web/admin_api.go` — Modify. Swap the `/about/commits` route for `/about/releases` (Task 2).
- `docker/card/admin-ui/src/pages/about.tsx` — Modify (full replacement). "Recent Releases" card + releases query + inline notes renderer (Task 3).
- `.github/workflows/ci.yml` — Modify. Add the `release` job (Task 4).
- `docker/card/build/build.go` + `CLAUDE.md` — Modify. Version bump + doc sync (Task 5).

---

## Task 1: Release-range selection helper (`selectReleases`)

**Files:**
- Modify: `docker/card/web/admin_api_about.go` (add helpers; also add `sort` to imports)
- Test: `docker/card/web/admin_api_about_test.go` (add `TestSelectReleases`)

**Interfaces:**
- Consumes: `CompareVersions(current, latest string) int` from `web/update.go`.
- Produces:
  - `func isVersion(s string) bool` — true iff `s` matches `^\d+\.\d+\.\d+$`.
  - `func selectReleases(running, latest string, versions []string) []string` — returns the subset of `versions` (each a bare `X.Y.Z`) to display, sorted newest-first. Range: `running <= ver <= upper`, where `upper = latest` when `latest` is a valid version greater than `running`, else `upper = running`. Non-version entries in `versions` are skipped. Returns `nil` when `running` is empty.

- [ ] **Step 1: Write the failing test**

Add to `docker/card/web/admin_api_about_test.go`:

```go
func TestSelectReleases(t *testing.T) {
	tests := []struct {
		name     string
		running  string
		latest   string
		versions []string
		want     []string
	}{
		{
			name:     "up to date shows only running",
			running:  "0.22.0",
			latest:   "0.22.0",
			versions: []string{"0.22.0", "0.21.0", "0.20.0"},
			want:     []string{"0.22.0"},
		},
		{
			name:     "behind shows running through latest, descending",
			running:  "0.20.0",
			latest:   "0.22.0",
			versions: []string{"0.19.0", "0.22.0", "0.20.0", "0.21.0"},
			want:     []string{"0.22.0", "0.21.0", "0.20.0"},
		},
		{
			name:     "running version absent from list",
			running:  "0.20.5",
			latest:   "0.22.0",
			versions: []string{"0.22.0", "0.21.0", "0.20.0"},
			want:     []string{"0.22.0", "0.21.0"},
		},
		{
			name:     "empty latest falls back to running only",
			running:  "0.22.0",
			latest:   "",
			versions: []string{"0.22.0", "0.21.0"},
			want:     []string{"0.22.0"},
		},
		{
			name:     "garbage latest falls back to running only",
			running:  "0.22.0",
			latest:   "not-a-version",
			versions: []string{"0.22.0", "0.21.0"},
			want:     []string{"0.22.0"},
		},
		{
			name:     "latest below running is ignored (no downgrade range)",
			running:  "0.22.0",
			latest:   "0.21.0",
			versions: []string{"0.22.0", "0.21.0"},
			want:     []string{"0.22.0"},
		},
		{
			name:     "non-version tags are skipped",
			running:  "0.21.0",
			latest:   "0.22.0",
			versions: []string{"0.22.0", "latest", "v0.21.0", "0.21.0"},
			want:     []string{"0.22.0", "0.21.0"},
		},
		{
			name:     "empty running returns nil",
			running:  "",
			latest:   "0.22.0",
			versions: []string{"0.22.0"},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectReleases(tt.running, tt.latest, tt.versions)
			if len(got) != len(tt.want) {
				t.Fatalf("selectReleases() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("selectReleases() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd docker/card && go test ./web/ -run TestSelectReleases -v`
Expected: FAIL — `undefined: selectReleases` (and `isVersion`).

- [ ] **Step 3: Add the helpers**

In `docker/card/web/admin_api_about.go`, add `"sort"` to the import block (alphabetical order: after `regexp`/before `strings`), and append these helpers (e.g. near the bottom, before `ansiColorMap`):

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd docker/card && go test ./web/ -run TestSelectReleases -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add docker/card/web/admin_api_about.go docker/card/web/admin_api_about_test.go
git commit -m "Add selectReleases range helper for About releases

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: `/about/releases` endpoint replaces `/about/commits`

**Files:**
- Modify: `docker/card/web/admin_api_about.go` (replace handler + remove dead code + imports)
- Modify: `docker/card/web/admin_api_about_test.go` (remove `TestParseVersionFromBuildGo`)
- Modify: `docker/card/web/admin_api.go` (route swap)

**Interfaces:**
- Consumes: `selectReleases`, `isVersion` (Task 1); `build.Version`; `writeJSON`; `CompareVersions`.
- Produces: `GET /admin/api/about/releases?latest=<X.Y.Z>` → JSON `{"releases": [{ "version": string, "name": string, "body": string, "date": string, "url": string, "isCurrent": bool }]}`, newest-first.

- [ ] **Step 1: Replace the handler and remove dead code**

In `docker/card/web/admin_api_about.go`:

(a) Remove `"encoding/base64"` and `"sync"` from the import block (no longer used after this task). Keep `card/build`, `encoding/json`, `html`, `io`, `net/http`, `regexp`, `sort`, `strings`, and the logrus import.

(b) Delete the entire `adminApiCommits` function.

(c) Delete `parseVersionFromBuildGo`, `buildVersionRegex`, the `commitVersionMu`/`commitVersionCache` `var` block, and the `fetchCommitVersion` function.

(d) Add the new handler (e.g. where `adminApiCommits` was):

```go
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
```

- [ ] **Step 2: Remove the obsolete test**

In `docker/card/web/admin_api_about_test.go`, delete the entire `TestParseVersionFromBuildGo` function (it references the now-removed `parseVersionFromBuildGo`). Leave `TestSelectReleases`. The file's only remaining import is `"testing"`.

- [ ] **Step 3: Swap the route**

In `docker/card/web/admin_api.go`, replace:

```go
		case path == "/admin/api/about/commits" && r.Method == "GET":
			app.adminApiAuth(app.adminApiCommits)(w, r)
```

with:

```go
		case path == "/admin/api/about/releases" && r.Method == "GET":
			app.adminApiAuth(app.adminApiReleases)(w, r)
```

- [ ] **Step 4: Verify it builds, vets, and tests pass**

Run: `cd docker/card && go vet ./web/ && go build ./... && go test ./web/ -count=1`
Expected: no vet errors, build succeeds, tests PASS. (If `go vet` flags an unused import, you missed removing `encoding/base64` or `sync` in Step 1(a).)

- [ ] **Step 5: Commit**

```bash
git add docker/card/web/admin_api_about.go docker/card/web/admin_api_about_test.go docker/card/web/admin_api.go
git commit -m "Replace About /commits with /releases endpoint

Single GitHub Releases API call (drops the per-commit N+1 fetch); filters
to the running..latest version range via selectReleases.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: About page — "Recent Releases" card

**Files:**
- Modify (full replacement): `docker/card/admin-ui/src/pages/about.tsx`

**Interfaces:**
- Consumes: `GET /about/releases?latest=<X.Y.Z>` JSON from Task 2 (`{ releases: [{ version, name, body, date, url, isCurrent }] }`).
- Produces: no downstream consumers (leaf page).

- [ ] **Step 1: Replace the file contents**

Overwrite `docker/card/admin-ui/src/pages/about.tsx` with:

```tsx
import { useQuery, useMutation } from "@tanstack/react-query";
import { apiFetch, apiPost } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ArrowUpCircle, Loader2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";

interface AboutData {
  version: string;
  buildDate: string;
  buildTime: string;
  latestVersion: string;
  updateAvailable: boolean;
}

interface LogsData {
  logs: string[];
}

interface Release {
  version: string;
  name: string;
  body: string;
  date: string;
  url: string;
  isCurrent: boolean;
}

interface ReleasesData {
  releases: Release[];
}

// linkify turns bare http(s) URLs in a line into anchor elements; other text is
// returned verbatim (React escapes it).
function linkify(text: string): ReactNode {
  const parts = text.split(/(https?:\/\/[^\s]+)/g);
  return parts.map((part, i) =>
    /^https?:\/\//.test(part) ? (
      <a
        key={i}
        href={part}
        target="_blank"
        rel="noopener noreferrer"
        className="text-primary hover:underline"
      >
        {part}
      </a>
    ) : (
      <span key={i}>{part}</span>
    ),
  );
}

// ReleaseNotes renders a release body: "- " lines become a bullet list, blank
// lines break groups, everything else is a paragraph. No markdown dependency.
function ReleaseNotes({ body }: { body: string }) {
  const elements: ReactNode[] = [];
  let bullets: string[] = [];

  const flush = () => {
    if (bullets.length > 0) {
      const items = bullets;
      elements.push(
        <ul
          key={elements.length}
          className="list-disc space-y-0.5 pl-5 text-sm"
        >
          {items.map((b, i) => (
            <li key={i}>{linkify(b)}</li>
          ))}
        </ul>,
      );
      bullets = [];
    }
  };

  for (const raw of body.split("\n")) {
    const line = raw.trimEnd();
    if (line.startsWith("- ")) {
      bullets.push(line.slice(2));
    } else if (line.trim() === "") {
      flush();
    } else {
      flush();
      elements.push(
        <p key={elements.length} className="text-sm text-muted-foreground">
          {linkify(line)}
        </p>,
      );
    }
  }
  flush();

  return <div className="space-y-2">{elements}</div>;
}

export function AboutPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["about"],
    queryFn: () => apiFetch<AboutData>("/about"),
  });

  const { data: logsData } = useQuery({
    queryKey: ["about-logs"],
    queryFn: () => apiFetch<LogsData>("/about/logs"),
  });

  const { data: releasesData } = useQuery({
    queryKey: ["about-releases", data?.latestVersion],
    queryFn: () =>
      apiFetch<ReleasesData>(
        `/about/releases?latest=${encodeURIComponent(data?.latestVersion ?? "")}`,
      ),
    enabled: !!data,
  });

  const [dialogOpen, setDialogOpen] = useState(false);

  const [updating, setUpdating] = useState(false);

  const triggerUpdate = useMutation({
    mutationFn: () => apiPost("/about/update"),
    onSettled: () => {
      setDialogOpen(false);
      setUpdating(true);
      toast.success("Update triggered — restarting containers…");
      // Poll until the server comes back with a new version
      const poll = setInterval(async () => {
        try {
          const res = await fetch("/admin/api/about");
          if (res.ok) {
            clearInterval(poll);
            window.location.reload();
          }
        } catch {
          // server still restarting
        }
      }, 3000);
      // Stop polling after 2 minutes
      setTimeout(() => clearInterval(poll), 120_000);
    },
  });

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">About</h1>
        <div className="h-48 animate-pulse rounded-lg bg-muted" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">About</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Software</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableBody>
              <TableRow>
                <TableCell className="font-medium">Version</TableCell>
                <TableCell className="font-mono">{data.version}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Build Date</TableCell>
                <TableCell>{data.buildDate}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Build Time</TableCell>
                <TableCell>{data.buildTime}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Latest Version</TableCell>
                <TableCell>
                  <span className="font-mono">
                    {data.latestVersion || "unable to check"}
                  </span>
                  {data.updateAvailable && (
                    <Badge variant="default" className="ml-2">
                      Update available
                    </Badge>
                  )}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>

          {data.updateAvailable && !updating && (
            <div className="mt-4">
              <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogTrigger asChild>
                  <Button>
                    <ArrowUpCircle className="mr-2 h-4 w-4" />
                    Update to {data.latestVersion}
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Confirm Update</DialogTitle>
                    <DialogDescription>
                      Pull latest images and restart containers?
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <Button
                      variant="outline"
                      onClick={() => setDialogOpen(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      onClick={() => triggerUpdate.mutate()}
                      disabled={triggerUpdate.isPending}
                    >
                      {triggerUpdate.isPending ? "Updating..." : "Update"}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          )}

          {updating && (
            <div className="mt-4 flex items-center gap-3 rounded-lg border p-4">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              <div>
                <p className="font-medium">Updating…</p>
                <p className="text-sm text-muted-foreground">
                  Pulling images and restarting containers. This page will
                  reload automatically.
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {logsData && logsData.logs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Recent Logs</CardTitle>
          </CardHeader>
          <CardContent>
            <pre
              className="overflow-x-auto rounded-md bg-muted p-3 text-xs leading-relaxed"
              dangerouslySetInnerHTML={{ __html: logsData.logs.join("\n") }}
            />
          </CardContent>
        </Card>
      )}

      {releasesData && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Recent Releases</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {releasesData.releases.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No release notes available.
              </p>
            ) : (
              releasesData.releases.map((rel) => (
                <div key={rel.version} className="space-y-2">
                  <div className="flex items-baseline gap-2">
                    <a
                      href={rel.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono text-sm font-medium hover:underline"
                    >
                      v{rel.version}
                    </a>
                    {rel.isCurrent && (
                      <Badge variant="secondary" className="text-xs">
                        Current
                      </Badge>
                    )}
                    {rel.date && (
                      <span className="text-xs text-muted-foreground">
                        {new Date(rel.date).toLocaleDateString(undefined, {
                          year: "numeric",
                          month: "short",
                          day: "numeric",
                        })}
                      </span>
                    )}
                  </div>
                  {rel.body.trim() && <ReleaseNotes body={rel.body} />}
                </div>
              ))
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Type-check and build the frontend**

Run:
```bash
export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
cd docker/card/admin-ui && npm run build
```
Expected: build succeeds with no TypeScript errors. (`npm run build` runs `tsc` then `vite build`.)

- [ ] **Step 3: Commit**

```bash
git add docker/card/admin-ui/src/pages/about.tsx
git commit -m "About: show Recent Releases instead of Recent Commits

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: CI — auto-publish a GitHub Release on version bump

**Files:**
- Modify: `.github/workflows/ci.yml` (append a `release` job)

**Interfaces:**
- Consumes: `docker` job success; `secrets.GITHUB_TOKEN`; tags `v*` in repo history.
- Produces: a git tag `v{version}` + GitHub Release whenever `build/build.go`'s version changes on `main`.

- [ ] **Step 1: Add the `release` job**

Append to `.github/workflows/ci.yml` (after the `docker` job, same indentation level — a new top-level entry under `jobs:`):

```yaml
  release:
    runs-on: ubuntu-latest
    needs: [docker]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Publish GitHub Release on version bump
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          V=$(grep -oP 'Version string = "\K[0-9]+\.[0-9]+\.[0-9]+' docker/card/build/build.go)
          echo "Version in build.go: $V"

          if git rev-parse -q --verify "refs/tags/v$V" >/dev/null; then
            echo "Tag v$V already exists — nothing to release."
            exit 0
          fi

          PREV=$(git tag -l 'v*' --sort=-v:refname | head -n1)
          echo "Previous release tag: ${PREV:-(none)}"
          if [ -n "$PREV" ]; then
            RANGE="$PREV..HEAD"
          else
            RANGE="HEAD"
          fi

          # Tidied notes: non-merge commit subjects, version-bump lines dropped.
          mapfile -t SUBJECTS < <(git log --no-merges --pretty=format:'%s' $RANGE \
            | grep -viE '^(bump|chore: ?bump).*version|^version bump|^bump to v?[0-9]' || true)

          TOTAL=${#SUBJECTS[@]}
          LIMIT=30
          BODY=""
          COUNT=0
          for s in "${SUBJECTS[@]}"; do
            [ "$COUNT" -ge "$LIMIT" ] && break
            BODY+="- $s"$'\n'
            COUNT=$((COUNT + 1))
          done
          if [ "$TOTAL" -gt "$LIMIT" ]; then
            REMAIN=$((TOTAL - LIMIT))
            BODY+=$'\n'"…and $REMAIN earlier commits."$'\n'
            if [ -n "$PREV" ]; then
              BODY+=$'\n'"Full changelog: https://github.com/boltcard/hub/compare/$PREV...v$V"$'\n'
            fi
          fi
          if [ -z "$BODY" ]; then
            BODY="Release v$V"
          fi

          echo "----- release notes -----"
          printf '%s\n' "$BODY"
          echo "-------------------------"

          gh release create "v$V" --target "$GITHUB_SHA" --title "v$V" --notes "$BODY"
```

- [ ] **Step 2: Validate the workflow YAML**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('YAML OK')"`
Expected: `YAML OK`. (If `python3`/`yaml` is unavailable, instead confirm the `release:` block is indented identically to the sibling `build:`/`docker:` jobs and that `permissions:` sits under `release:`, not at workflow scope.)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "CI: publish GitHub Release on version bump

New release job tags v{version} and publishes notes built from non-merge
commit subjects since the previous tag (capped at 30). Idempotent: skips if
the tag already exists. Job-scoped contents:write keeps the workflow default
at contents:read.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Version bump + docs

**Files:**
- Modify: `docker/card/build/build.go`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: nothing.
- Produces: version `0.22.0`.

- [ ] **Step 1: Bump the version**

In `docker/card/build/build.go`, change:

```go
var Version string = "0.21.0"
```
to:
```go
var Version string = "0.22.0"
```

- [ ] **Step 2: Sync CLAUDE.md**

In `CLAUDE.md`, change the `build/` bullet:

```
- `build/` — Version string (currently "0.21.0"), date/time injected at build
```
to:
```
- `build/` — Version string (currently "0.22.0"), date/time injected at build
```

- [ ] **Step 3: Full verification (Go + frontend)**

Run:
```bash
cd docker/card && go vet ./... && go build -o /tmp/app && go test -race -count=1 ./...
```
Expected: vet clean, build succeeds, all tests PASS.

Then:
```bash
export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
cd docker/card/admin-ui && npm run build
```
Expected: frontend build succeeds.

- [ ] **Step 4: Commit**

```bash
git add docker/card/build/build.go CLAUDE.md
git commit -m "Bump version to 0.22.0

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Final verification checklist

- [ ] `cd docker/card && go vet ./... && go test -race -count=1 ./...` — clean.
- [ ] `npm run build` in `docker/card/admin-ui` — clean.
- [ ] `grep -rn "adminApiCommits\|fetchCommitVersion\|commitVersionCache\|parseVersionFromBuildGo\|/about/commits" docker/card/` — no matches (dead code fully removed).
- [ ] `build/build.go` says `0.22.0`; `CLAUDE.md` build line says `0.22.0`.
- [ ] `.github/workflows/ci.yml` has a `release` job with job-level `permissions: contents: write`; workflow-level `permissions:` is still `contents: read`.
