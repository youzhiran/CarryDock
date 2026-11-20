package updater

import "testing"

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		local, remote string
		want          bool
	}{
		{"0.8.4+104", "v0.8.5+105", true},
		{"0.9.0+105", "v0.8.4+104", false},
		{"0.9.0+105", "v0.9.0+105", false},
		{"0.8.4", "v0.9.0", true},
		{"0.9.0", "v0.8.9", false},
		{"1.0.0", "v1.0.1", true},
	}
	for _, tc := range cases {
		if got := IsNewerVersion(tc.local, tc.remote); got != tc.want {
			t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", tc.local, tc.remote, got, tc.want)
		}
	}
}
