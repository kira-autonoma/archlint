package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mshogin/archlint/internal/analyzer"
	"github.com/mshogin/archlint/internal/mcp"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard [directory]",
	Short: "Create an adaptive .archlint.yaml for gradual adoption",
	Long: `Scan a project, count violations, and generate an .archlint.yaml that
surfaces only the top 5 most critical violations. This lets you adopt archlint
incrementally: fix those 5, then run "archlint onboard ." again to tighten.

Examples:
  archlint onboard .
  archlint onboard ./myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runOnboard,
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}

// onboardMaxViolations is the target number of violations to surface.
const onboardMaxViolations = 5

// violationSeverity returns a priority for each violation kind.
// Lower number = more critical = reported first.
func violationSeverity(kind string) int {
	switch kind {
	case "circular-dependency":
		return 1
	case "high-efferent-coupling":
		return 2
	case "srp":
		return 3
	case "dip":
		return 4
	case "isp":
		return 5
	case "god-class":
		return 6
	case "hub-node":
		return 7
	case "feature-envy":
		return 8
	case "shotgun-surgery":
		return 9
	default:
		return 10
	}
}

// OnboardThresholds represents the thresholds section of .archlint.yaml.
type OnboardThresholds struct {
	SRPMethods   int `yaml:"srp_methods"`
	SRPFields    int `yaml:"srp_fields"`
	ISPMethods   int `yaml:"isp_methods"`
	GodMethods   int `yaml:"god_methods"`
	GodFields    int `yaml:"god_fields"`
	GodFanOut    int `yaml:"god_fan_out"`
	HubFanOut    int `yaml:"hub_fan_out"`
	Shotgun      int `yaml:"shotgun_dependents"`
	HighCoupling int `yaml:"high_coupling"`
}

// OnboardConfig represents the full .archlint.yaml file.
type OnboardConfig struct {
	Version    string            `yaml:"version"`
	Thresholds OnboardThresholds `yaml:"thresholds"`
	Suppress   []string          `yaml:"suppress,omitempty"`
	Onboard    *OnboardPlan      `yaml:"onboard,omitempty"`
}

// OnboardPlan stores iteration metadata.
type OnboardPlan struct {
	CurrentViolations int    `yaml:"current_violations"`
	TotalViolations   int    `yaml:"total_violations"`
	Iteration         int    `yaml:"iteration"`
	Strategy          string `yaml:"strategy"`
}

// defaultThresholds returns the built-in default thresholds.
func defaultThresholds() OnboardThresholds {
	return OnboardThresholds{
		SRPMethods:   7,
		SRPFields:    10,
		ISPMethods:   5,
		GodMethods:   15,
		GodFields:    20,
		GodFanOut:    10,
		HubFanOut:    15,
		Shotgun:      5,
		HighCoupling: 10,
	}
}

func runOnboard(_ *cobra.Command, args []string) error {
	codeDir := args[0]

	if _, err := os.Stat(codeDir); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", errDirNotExist, codeDir)
	}

	// Step 1: Run analysis with default config.
	a := analyzer.NewGoAnalyzer()

	graph, err := a.Analyze(codeDir)
	if err != nil {
		return fmt.Errorf("analysis error: %w", err)
	}

	// Step 2: Collect all violations (same as check command).
	violations := mcp.DetectAllViolations(graph)

	allMetrics := mcp.ComputeAllFileMetrics(a, graph)
	for _, m := range allMetrics {
		violations = append(violations, m.SRPViolations...)
		violations = append(violations, m.DIPViolations...)
		violations = append(violations, m.ISPViolations...)

		for _, gc := range m.GodClasses {
			violations = append(violations, mcp.Violation{
				Kind:    "god-class",
				Message: fmt.Sprintf("God class detected: %s", gc),
				Target:  gc,
			})
		}

		for _, hub := range m.HubNodes {
			violations = append(violations, mcp.Violation{
				Kind:    "hub-node",
				Message: fmt.Sprintf("Hub node detected: %s", hub),
				Target:  hub,
			})
		}

		for _, fe := range m.FeatureEnvy {
			violations = append(violations, mcp.Violation{
				Kind:    "feature-envy",
				Message: fmt.Sprintf("Feature envy: %s", fe),
				Target:  fe,
			})
		}

		for _, ss := range m.ShotgunSurgery {
			violations = append(violations, mcp.Violation{
				Kind:    "shotgun-surgery",
				Message: fmt.Sprintf("Shotgun surgery risk: %s", ss),
				Target:  ss,
			})
		}
	}

	totalViolations := len(violations)

	// Step 3: Sort by severity (most critical first).
	sort.Slice(violations, func(i, j int) bool {
		si := violationSeverity(violations[i].Kind)
		sj := violationSeverity(violations[j].Kind)

		if si != sj {
			return si < sj
		}

		return violations[i].Target < violations[j].Target
	})

	// Step 4: Determine thresholds.
	cfg := OnboardConfig{
		Version:    "1",
		Thresholds: defaultThresholds(),
	}

	surfaced := totalViolations
	if totalViolations > onboardMaxViolations {
		// Keep only top 5 — suppress the rest by kind.
		kept := violations[:onboardMaxViolations]
		suppressed := violations[onboardMaxViolations:]

		// Find which kinds to suppress entirely.
		keptKinds := make(map[string]bool)
		for _, v := range kept {
			keptKinds[v.Kind] = true
		}

		suppressKinds := make(map[string]bool)

		for _, v := range suppressed {
			if !keptKinds[v.Kind] {
				suppressKinds[v.Kind] = true
			}
		}

		for kind := range suppressKinds {
			cfg.Suppress = append(cfg.Suppress, kind)
		}

		sort.Strings(cfg.Suppress)

		// For kinds that appear in both kept and suppressed, raise thresholds.
		// We raise thresholds significantly to suppress most violations of that kind.
		for _, v := range suppressed {
			if keptKinds[v.Kind] {
				raiseThreshold(&cfg.Thresholds, v.Kind)
			}
		}

		surfaced = onboardMaxViolations
	}

	cfg.Onboard = &OnboardPlan{
		CurrentViolations: surfaced,
		TotalViolations:   totalViolations,
		Iteration:         1,
		Strategy:          "Fix current violations, then run 'archlint onboard .' again to tighten thresholds.",
	}

	// Step 5: Write .archlint.yaml.
	absDir, err := filepath.Abs(codeDir)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}

	configPath := filepath.Join(absDir, ".archlint.yaml")

	if err := writeOnboardConfig(configPath, &cfg); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	// Step 6: Print summary.
	fmt.Printf("Created .archlint.yaml (%d violations out of %d total).\n", surfaced, totalViolations)

	if totalViolations > onboardMaxViolations {
		fmt.Println()
		fmt.Println("Iteration plan:")
		fmt.Printf("  Iteration 1: %d violations to fix now\n", surfaced)
		fmt.Println("  Next: fix these, then run `archlint onboard .` to tighten thresholds")
		fmt.Println("  Repeat until all violations are resolved")
	}

	fmt.Printf("\nRun `archlint check %s` to see current violations.\n", codeDir)

	return nil
}

// raiseThreshold doubles the relevant threshold for a given violation kind
// to suppress some violations of that kind while keeping the most critical.
func raiseThreshold(t *OnboardThresholds, kind string) {
	const factor = 2

	switch kind {
	case "high-efferent-coupling":
		t.HighCoupling *= factor
	case "srp":
		t.SRPMethods *= factor
		t.SRPFields *= factor
	case "dip":
		// DIP violations are structural, no simple threshold to raise.
	case "isp":
		t.ISPMethods *= factor
	case "god-class":
		t.GodMethods *= factor
		t.GodFields *= factor
		t.GodFanOut *= factor
	case "hub-node":
		t.HubFanOut *= factor
	case "shotgun-surgery":
		t.Shotgun *= factor
	}
}

func writeOnboardConfig(path string, cfg *OnboardConfig) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing file: %v\n", closeErr)
		}
	}()

	// Write a header comment.
	header := "# Generated by archlint onboard. Adjust thresholds as you fix violations.\n"

	if _, err := file.WriteString(header); err != nil {
		return fmt.Errorf("cannot write header: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)

	defer func() {
		if closeErr := encoder.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing encoder: %v\n", closeErr)
		}
	}()

	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}

	return nil
}
