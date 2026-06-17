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
