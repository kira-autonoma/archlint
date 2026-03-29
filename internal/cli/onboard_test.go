package cli

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestViolationSeverity(t *testing.T) {
	tests := []struct {
		kind     string
		expected int
	}{
		{"circular-dependency", 1},
		{"high-efferent-coupling", 2},
		{"srp", 3},
		{"dip", 4},
		{"isp", 5},
		{"god-class", 6},
		{"hub-node", 7},
		{"feature-envy", 8},
		{"shotgun-surgery", 9},
		{"unknown-kind", 10},
	}

	for _, tt := range tests {
		got := violationSeverity(tt.kind)
		if got != tt.expected {
			t.Errorf("violationSeverity(%q) = %d, want %d", tt.kind, got, tt.expected)
		}
	}
}

func TestRaiseThreshold(t *testing.T) {
	th := defaultThresholds()
	original := defaultThresholds()

	raiseThreshold(&th, "high-efferent-coupling")

	if th.HighCoupling != original.HighCoupling*2 {
		t.Errorf("HighCoupling = %d, want %d", th.HighCoupling, original.HighCoupling*2)
	}

	raiseThreshold(&th, "srp")

	if th.SRPMethods != original.SRPMethods*2 {
		t.Errorf("SRPMethods = %d, want %d", th.SRPMethods, original.SRPMethods*2)
	}

	if th.SRPFields != original.SRPFields*2 {
		t.Errorf("SRPFields = %d, want %d", th.SRPFields, original.SRPFields*2)
	}

	raiseThreshold(&th, "god-class")

	if th.GodMethods != original.GodMethods*2 {
		t.Errorf("GodMethods = %d, want %d", th.GodMethods, original.GodMethods*2)
	}
}

func TestWriteOnboardConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".archlint.yaml")

	cfg := &OnboardConfig{
		Version:    "1",
		Thresholds: defaultThresholds(),
		Suppress:   []string{"feature-envy"},
		Onboard: &OnboardPlan{
			CurrentViolations: 5,
			TotalViolations:   20,
			Iteration:         1,
			Strategy:          "Fix current violations, then run 'archlint onboard .' again to tighten thresholds.",
		},
	}

	if err := writeOnboardConfig(configPath, cfg); err != nil {
		t.Fatalf("writeOnboardConfig() error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("cannot read config file: %v", err)
	}

	content := string(data)

	// Check the header is present.
	if len(content) < 10 {
		t.Fatal("config file is too short")
	}

	// Parse back and verify.
	var parsed OnboardConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("cannot parse written config: %v", err)
	}

	if parsed.Version != "1" {
		t.Errorf("Version = %q, want %q", parsed.Version, "1")
	}

	if parsed.Thresholds.SRPMethods != 7 {
		t.Errorf("SRPMethods = %d, want %d", parsed.Thresholds.SRPMethods, 7)
	}

	if len(parsed.Suppress) != 1 || parsed.Suppress[0] != "feature-envy" {
		t.Errorf("Suppress = %v, want [feature-envy]", parsed.Suppress)
	}

	if parsed.Onboard == nil {
		t.Fatal("Onboard plan is nil")
	}

	if parsed.Onboard.CurrentViolations != 5 {
		t.Errorf("CurrentViolations = %d, want %d", parsed.Onboard.CurrentViolations, 5)
	}

	if parsed.Onboard.TotalViolations != 20 {
		t.Errorf("TotalViolations = %d, want %d", parsed.Onboard.TotalViolations, 20)
	}
}

func TestDefaultThresholds(t *testing.T) {
	th := defaultThresholds()

	if th.SRPMethods != 7 {
		t.Errorf("SRPMethods = %d, want 7", th.SRPMethods)
	}

	if th.HighCoupling != 10 {
		t.Errorf("HighCoupling = %d, want 10", th.HighCoupling)
	}

	if th.GodMethods != 15 {
		t.Errorf("GodMethods = %d, want 15", th.GodMethods)
	}
}
