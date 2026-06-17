# About page: Recent Releases (release notes) — design

Date: 2026-06-17
Branch: `claude/about-release-notes`
Base: `origin/main` @ `0fcad68` (includes PR #40 per-commit version markers)

## Problem

The About page shows a "Recent Commits" list with a muted `vX.Y.Z` marker per
commit (PR #40). It reads like a raw, untidied changelog — merge-commit noise,
developer phrasing, and no grouping into releases. The user wants proper,
user-readable release notes grouped by release, scoped to the gap between the
hub's running version and the latest available version.

## Goals

1. Publish real **GitHub Releases** automatically when the app version is
   bumped, with notes sourced from the commits in that release, tidied.
2. Replace the About page's "Recent Commits" card with **"Recent Releases"**,
   sourced from the GitHub Releases API.
3. Show the release notes **between the running version and the latest
   available version** (inclusive of both ends). When up to date, show only the
   running version's notes.

## Non-goals

- No AI/LLM summarisation of notes (CI can't call a model). "Tidied" means
  merge-commit noise stripped + version-bump lines dropped + bullet formatting.
- No backfilling of historical releases (the gap from v0.10.0 is handled by a
  one-time capped "catch-up" release, not by reconstructing per-version tags).
- No new markdown dependency in the frontend.

## Decisions (from brainstorming)

- **Notes source:** auto-publish GitHub Releases in CI on version bump.
- **Tidying method:** commit subjects with merge commits stripped
  (`git log --no-merges`), version-bump lines filtered.
- **Up-to-date display:** show only the current (running) version's notes.

## Part 1 — CI: auto-publish a GitHub Release on version bump

File: `.github/workflows/ci.yml`. Add a `release` job.

- `needs: [docker]` — only release once images build/push succeeds.
- Gate: `if: github.event_name == 'push' && github.ref == 'refs/heads/main'`.
- **Job-level** `permissions: contents: write` (the workflow default stays
  `contents: read` — least privilege preserved; only this job can tag/release).
- `actions/checkout@v4` with `fetch-depth: 0` so full history + tags are present.
- Steps (shell):
  1. Extract `V` from `docker/card/build/build.go`
     (`grep -oP 'Version string = "\K[0-9]+\.[0-9]+\.[0-9]+'`).
  2. If tag `v$V` already exists (`git rev-parse -q --verify "refs/tags/v$V"`),
     **skip the rest** (idempotent: re-runs and merges without a version bump
     never create a duplicate release).
  3. Determine the previous release tag:
     `PREV=$(git tag -l 'v*' --sort=-v:refname | head -n1)`.
     - If empty (no tags at all), use the root commit as the range start.
  4. Build the notes body:
     - `git log --no-merges --pretty=format:'%s' "$PREV"..HEAD` (or whole
       history if no `PREV`).
     - Drop lines that are pure version bumps
       (grep -viE pattern: `^(bump|chore: bump).*version|^version bump|^bump to v?[0-9]`).
     - Prefix each remaining subject with `- `.
     - **Cap at 30 lines.** If more, keep the 30 most recent and append:
       `` and `N` earlier commits`` plus a blank line and
       `Full changelog: https://github.com/boltcard/hub/compare/$PREV...v$V`.
       (No `PREV` → omit the compare line.)
  5. `gh release create "v$V" --title "v$V" --notes "$BODY"`
     with `env: GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`.

Idempotency + the cap make the first ("catch-up") release readable rather than a
several-hundred-commit dump; every subsequent per-version release has a small,
clean span.

## Part 2 — Backend: `/about/releases` replaces `/about/commits`

Files: `docker/card/web/admin_api_about.go`, `docker/card/web/admin_api.go`.

- Remove `adminApiCommits` and the per-commit version machinery it relied on:
  `fetchCommitVersion`, `commitVersionMu`, `commitVersionCache`,
  `parseVersionFromBuildGo`, `buildVersionRegex`. The N+1 GitHub contents fetch
  is gone. (Keep `adminApiLogs`, `ansiToHTML`, `ansiColorMap`, `ansiRegex`.)
- Remove the `/admin/api/about/commits` route; add
  `GET /admin/api/about/releases` → `adminApiAuth(app.adminApiReleases)`.

`adminApiReleases`:

- `R := build.Version`. `L := strings.TrimSpace(r.URL.Query().Get("latest"))`
  — the frontend already has the latest version from `/about`, so it passes it
  through; this avoids a second Docker Hub round-trip. `L` is public release
  data used only to bound which already-public notes to show, so trusting the
  client param has no security impact.
- One GET to `https://api.github.com/repos/boltcard/hub/releases?per_page=100`
  (10s timeout, `Accept: application/vnd.github.v3+json`). Non-200 / error →
  return `{"releases": []}`.
- Decode `[]{ tag_name, name, body, published_at, html_url }`.
- Compute the upper bound:
  `upper := R; if L parses (3-part numeric) and CompareVersions(R, L) == 1 { upper = L }`
  (`CompareVersions(current, latest)` returns 1 when `latest > current`).
- For each release: parse the version from `tag_name` (strip a leading `v`;
  skip tags that aren't 3-part numeric). Include it when ver is in `[R, upper]`.
  With `CompareVersions(a, b)` returning `1` when `b > a`:
  - ver ≥ R ⇔ `CompareVersions(R, ver) >= 0`
  - ver ≤ upper ⇔ `CompareVersions(ver, upper) >= 0`
- Sort the included releases descending by version.
- Respond `{"releases": [ { version, name, body, date, url, isCurrent } ]}`
  where `isCurrent = (ver == R)`, `date = published_at`, `url = html_url`.

Extract the pure decision — "given R, L and a list of release versions, which
versions are shown and in what order" — into a small testable helper
(e.g. `selectReleases(running, latest string, versions []string) []string`)
so the range logic is unit-tested without hitting the network.

## Part 3 — Frontend: `about.tsx`

File: `docker/card/admin-ui/src/pages/about.tsx`.

- Replace the `Commit`/`CommitsData` types and the `about-commits` query with a
  `Release`/`ReleasesData` type and an `about-releases` query:
  `apiFetch<ReleasesData>(\`/about/releases?latest=${encodeURIComponent(data.latestVersion || "")}\`)`.
  Gate the query on `data` being loaded (it needs `latestVersion`).
- Rename the card title **"Recent Commits" → "Recent Releases"**.
- Render each release as a block:
  - Heading row: `v{version}` (mono) + formatted `date`; a `Current` badge when
    `isCurrent`; the heading links to `url` (release page) in a new tab.
  - Body: split on newlines; lines starting with `- ` become `<li>` in a `<ul>`;
    other non-empty lines become `<p>`; bare URLs are linkified. Plain text via
    React children (auto-escaped) — no `dangerouslySetInnerHTML`, no markdown lib.
- Empty state: if `releases.length === 0`, render a muted
  "No release notes available." line (or hide the card body) instead of the list.

## Part 4 — Version bump + docs

- `docker/card/build/build.go`: `0.21.0` → `0.22.0` (MINOR — new feature).
- `CLAUDE.md`: update the `build/` bullet's "currently 0.21.0" reference to
  `0.22.0`. Optionally note the new `release` CI job + `/about/releases` endpoint
  in the relevant sections.

## Testing

- **Go:** table-driven test for `selectReleases` covering:
  up-to-date (`L==R` → only R), behind (`R < L` → R..L inclusive, descending),
  running version has no release in the list (fewer/none returned),
  unparseable/empty `L` (falls back to only R), and tag strings with/without a
  `v` prefix. Run `cd docker/card && go test -race -count=1 ./web/`.
- **Frontend:** build-checked only (no FE test runner) — `npm run build` in
  `docker/card/admin-ui`.
- **CI release job:** not unit-tested (shell in the workflow); validated by
  the idempotency guard (safe to re-run) and reviewed manually.

## Edge cases

- GitHub API down / rate-limited → empty releases list → empty state.
- Hub running a version older than the first published release → range yields
  nothing → empty state (expected until releases accumulate).
- Docker Hub check failed (`L` empty) → treated as up-to-date → only R's notes.
- Tag `vX.Y.Z` exists but no matching version in build.go yet → handled by the
  CI "skip if tag exists" guard.
