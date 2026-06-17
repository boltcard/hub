package web

import "testing"

func TestParseVersionFromBuildGo(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "current format",
			content: "package build\n\nvar Version string = \"0.20.1\"\nvar Date string\nvar Time string\n",
			want:    "0.20.1",
		},
		{
			name:    "older multi-digit version",
			content: "var Version string = \"0.2.10\"\n",
			want:    "0.2.10",
		},
		{
			name:    "no version line",
			content: "package build\n\nvar Date string\n",
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseVersionFromBuildGo(tt.content); got != tt.want {
				t.Errorf("parseVersionFromBuildGo() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
