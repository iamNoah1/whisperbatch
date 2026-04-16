package transcriber

import "testing"

func TestSelectModel_VRAM(t *testing.T) {
	cases := []struct {
		vramMB float64
		want   string
	}{
		{10*1024 + 1, "large"},
		{10 * 1024, "large"},
		{5 * 1024, "medium"},
		{2 * 1024, "base"},
		{1 * 1024, "tiny"}, // VRAM too small, falls through to RAM; 1 GB RAM → tiny
	}
	for _, tc := range cases {
		// RAM set to 0 so only VRAM path (or tiny fallback) is exercised.
		name, _ := selectModel(0, tc.vramMB)
		if name != tc.want {
			t.Errorf("selectModel(ram=0, vram=%.0f) = %q, want %q", tc.vramMB, name, tc.want)
		}
	}
}

func TestSelectModel_RAM(t *testing.T) {
	cases := []struct {
		ramMB float64
		want  string
	}{
		{16*1024 + 1, "large"},
		{16 * 1024, "large"},
		{8 * 1024, "medium"},
		{4 * 1024, "base"},
		{2 * 1024, "tiny"},
		{0, "tiny"},
	}
	for _, tc := range cases {
		name, _ := selectModel(tc.ramMB, 0)
		if name != tc.want {
			t.Errorf("selectModel(ram=%.0f, vram=0) = %q, want %q", tc.ramMB, name, tc.want)
		}
	}
}
