package scripts

import "testing"

func TestNormalizeHelmVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "empty version stays empty", version: "", want: ""},
		{name: "latest becomes empty", version: "latest", want: ""},
		{name: "latest is case-insensitive", version: "LATEST", want: ""},
		{name: "surrounding whitespace trimmed", version: "  1.2.3  ", want: "1.2.3"},
		{name: "explicit version preserved", version: "1.2.3", want: "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeHelmVersion(tt.version); got != tt.want {
				t.Fatalf("normalizeHelmVersion(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}
