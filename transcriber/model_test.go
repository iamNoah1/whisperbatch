package transcriber

import "testing"

func TestSelectModel_VRAM(t *testing.T) {
	cases := []struct {
		vramMB float64
		want   string
	}{
		{6*1024 + 1, "large-v3"},
		{6 * 1024, "large-v3"},
		{3 * 1024, "medium"},
		{2 * 1024, "small"},
		{1 * 1024, "base"},
		// Below 1 GB VRAM: falls through to the RAM check; with RAM=0 we land in
		// the < 1 GB tier → tiny.
		{512, "tiny"},
	}
	for _, tc := range cases {
		// RAM set to 0 so only VRAM path (or tiny fallback) is exercised.
		name, _ := selectModel(0, tc.vramMB)
		if name != tc.want {
			t.Errorf("selectModel(ram=0, vram=%.0f) = %q, want %q", tc.vramMB, name, tc.want)
		}
	}
}

// TestSelectModel_VRAM_fallthrough verifies that when VRAM is present but below
// 1 GB, the RAM tier is used rather than defaulting to tiny unconditionally.
func TestSelectModel_VRAM_fallthrough(t *testing.T) {
	cases := []struct {
		vramMB float64
		ramMB  float64
		want   string
	}{
		{512, 8 * 1024, "large-v3"}, // small GPU, lots of RAM
		{512, 4 * 1024, "medium"},
		{512, 2 * 1024, "small"},
		{512, 1 * 1024, "base"},
		{512, 0, "tiny"},
	}
	for _, tc := range cases {
		name, _ := selectModel(tc.ramMB, tc.vramMB)
		if name != tc.want {
			t.Errorf("selectModel(ram=%.0f, vram=%.0f) = %q, want %q", tc.ramMB, tc.vramMB, name, tc.want)
		}
	}
}

func TestSelectModel_RAM(t *testing.T) {
	cases := []struct {
		ramMB float64
		want  string
	}{
		{8*1024 + 1, "large-v3"},
		{8 * 1024, "large-v3"},
		{4 * 1024, "medium"},
		{2 * 1024, "small"},
		{1 * 1024, "base"},
		{512, "tiny"},
		{0, "tiny"},
	}
	for _, tc := range cases {
		name, _ := selectModel(tc.ramMB, 0)
		if name != tc.want {
			t.Errorf("selectModel(ram=%.0f, vram=0) = %q, want %q", tc.ramMB, name, tc.want)
		}
	}
}
