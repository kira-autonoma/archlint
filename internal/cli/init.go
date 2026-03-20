package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Output setup context for AI agents configuring archlint",
	Long: `Outputs a structured context document describing archlint capabilities,
all available rules and categories, configuration format, and suggested
questions to ask the user. Designed to be consumed by an AI agent that
will then configure archlint for a specific project.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	fmt.Print(initContext)
	return nil
}

const initContext = `# archlint — AI Agent Setup Context

## What is archlint

archlint is an architecture linter for Go projects. It performs static analysis of Go source code via AST parsing, builds structural and behavioral graphs, and validates architecture against configurable rules.

## Available Commands

- ` + "`archlint collect <dir> -o <file.yaml>`" + ` — Analyze Go source code and build a structural graph (packages, types, functions, methods and their relationships). Output is a YAML file with components and links.
- ` + "`archlint callgraph <dir> --entry <entry_point>`" + ` — Build a call graph from a specific entry point using static AST analysis. Supports interface resolution, goroutine detection, defer tracking.
- ` + "`archlint callgraph <dir> --bpmn-contexts <file.yaml>`" + ` — Build call graphs for all entry points defined in a BPMN context mapping file.
- ` + "`archlint bpmn <file.bpmn> -o <file.yaml>`" + ` — Parse a BPMN 2.0 XML file into a structured process graph.
- ` + "`archlint validate <architecture.yaml>`" + ` — Validate an architecture graph against rules defined in .archlint.yaml.
- ` + "`archlint init`" + ` — This command. Outputs setup context for AI agents.

## Configuration File: .archlint.yaml

The configuration file goes in the project root as ` + "`.archlint.yaml`" + `. It defines which rules to enable and their thresholds.

### Format

` + "```yaml" + `
rules:
  <rule_name>:
    enabled: true|false          # Whether the rule is active
    error_on_violation: true|false  # true = fail the check, false = warn only
    threshold: <number>          # Rule-specific threshold (not all rules have one)
    exclude:                     # Glob patterns for components to skip
      - "cmd/*"
      - "pkg/models"
    params:                      # Rule-specific parameters (not all rules have them)
      key: value
` + "```" + `

## Rule Categories

### 1. Core Graph Rules (start here)
Fundamental structural checks. These catch the most impactful issues.

| Rule | What it checks | Key params |
|------|---------------|------------|
| dag_check | Dependency graph is a DAG (no cycles) | — |
| max_fan_out | Max outgoing dependencies per component | threshold (default: 5) |
| modularity | Graph modularity score | threshold (default: 0.3) |
| coupling | Afferent/efferent coupling per component | ca_threshold, ce_threshold (default: 10) |
| instability | Robert Martin's instability metric (Ce/(Ca+Ce)) | — |
| orphan_nodes | Components with no connections | — |
| strongly_connected_components | Circular dependency clusters | max_size (default: 1) |
| graph_depth | Max depth of dependency chain | threshold (default: 10) |
| hub_nodes | Components with too many connections | threshold (default: 10) |

### 2. Layer Architecture Rules
Enforce layered architecture conventions.

| Rule | What it checks | Key params |
|------|---------------|------------|
| layer_violations | Dependencies only flow downward through layers | layers: map of layer→level |
| layer_traversal | No skipping layers in dependency chain | layers: map of layer→level |
| forbidden_dependencies | Specific pairs that must not depend on each other | rules: [{from, to}] |
| inward_dependencies | Dependencies point inward (clean architecture) | — |

Default layer ordering (customize to your project):
` + "```" + `
cmd: 0, api/handler/controller: 1, service/usecase/internal: 2,
domain/entity: 3, repository/storage: 4, infrastructure: 5, pkg: 6, model: 7
` + "```" + `

### 3. SOLID Rules
Object-oriented design principle checks.

| Rule | What it checks | Key params |
|------|---------------|------------|
| single_responsibility | Components don't have too many responsibilities | threshold (default: 3) |
| open_closed | Extension points vs modification surface | threshold (default: 0.2) |
| liskov_substitution | Interface implementation consistency | threshold (default: 3) |
| interface_segregation | Interfaces aren't too large | threshold (default: 5) |
| dependency_inversion | High-level modules depend on abstractions | threshold (default: 0.3) |

### 4. Code Smell / Pattern Rules
Detect common architectural anti-patterns.

| Rule | What it checks | Key params |
|------|---------------|------------|
| god_class | Types with too many methods/dependencies | max_methods: 20, max_dependencies: 15 |
| feature_envy | Components using other modules' internals more than their own | threshold (default: 0.5) |
| shotgun_surgery | Changes that would ripple across many components | threshold (default: 10) |
| data_clumps | Groups of data that appear together repeatedly | threshold (default: 3) |
| divergent_change | Components changed for multiple unrelated reasons | threshold (default: 3) |
| lazy_class | Components with too little behavior to justify existence | min_methods: 2, min_dependencies: 1 |
| middle_man | Components that just delegate everything | — |
| speculative_generality | Unused abstractions | — |

### 5. Architecture Style Rules
Validate adherence to specific architectural patterns.

| Rule | What it checks | Key params |
|------|---------------|------------|
| ports_adapters | Hexagonal architecture port/adapter boundaries | — |
| domain_isolation | Domain layer is free of infrastructure imports | forbidden_patterns: [database, db, sql, http, grpc, redis] |
| bounded_context_leakage | Bounded contexts don't leak into each other | — |
| use_case_purity | Use cases don't contain infrastructure concerns | — |
| dto_boundaries | DTOs used at boundaries, not passed through layers | — |

### 6. Quality & Security Rules
Production-readiness checks.

| Rule | What it checks | Key params |
|------|---------------|------------|
| blast_radius | How much of the system a single component failure affects | threshold (default: 0.3) |
| hotspot_detection | Components with excessive coupling (change risk) | threshold (default: 10) |
| sensitive_data_flow | Sensitive data doesn't flow to unsafe components | — |
| auth_boundary | Authentication checks at system boundaries | — |
| input_validation_layer | Input validation at entry points | — |
| single_point_of_failure | Components that too many contexts depend on | min_contexts: 3 |
| deprecated_usage | Usage of deprecated/legacy components | patterns: [deprecated, legacy, old, obsolete] |

### 7. Graph Theory / Metrics Rules
Advanced structural metrics.

| Rule | What it checks | Key params |
|------|---------------|------------|
| betweenness_centrality | Bottleneck components (high betweenness) | threshold (default: 0.3) |
| pagerank | Most "important" components by link structure | threshold (default: 0.1) |
| edge_density | Graph isn't too sparse or too dense | min_threshold: 0.01, max_threshold: 0.3 |
| graph_diameter | Longest shortest path in the graph | threshold (default: 10) |
| gini_coefficient | Inequality in dependency distribution | threshold (default: 0.6) |
| algebraic_connectivity | How well-connected the graph is | threshold (default: 0.1) |
| abstractness | Balance of abstract vs concrete components | min_threshold: 0.1, max_threshold: 0.8 |
| distance_from_main_sequence | Robert Martin's D metric (abstractness + instability = 1) | threshold (default: 0.5) |
| component_distance | Max shortest-path distance between components | threshold (default: 5) |
| component_complexity | Structural complexity per component | threshold (default: 50) |

### 8. Research / Topological Rules
Experimental rules based on algebraic topology and advanced graph theory. Most are warning-only by default. Enable selectively if you understand the metric. These include: persistent_homology, betti_numbers, spectral_gap, ricci_flow, heat_diffusion, markov_properties, and ~80 more. See .archlint.yaml for the full list.

## Recommended Setup Profiles

### Minimal (for getting started or small projects)
Enable only core graph rules. Good for initial adoption.

Rules to enable: dag_check, max_fan_out, coupling, orphan_nodes, strongly_connected_components.

### Standard (recommended for most projects)
Core + layers + SOLID + key patterns. Catches real issues without noise.

Rules to enable: all from Minimal, plus: layer_violations, forbidden_dependencies, single_responsibility, interface_segregation, dependency_inversion, god_class, feature_envy, shotgun_surgery, blast_radius, hotspot_detection, domain_isolation.

### Strict (for mature codebases with clean architecture)
Standard + full architecture validation + quality/security + graph metrics.

Rules to enable: all from Standard, plus: all layer rules, all SOLID rules, all pattern rules, all architecture style rules, all quality rules, key graph metrics (betweenness_centrality, pagerank, gini_coefficient, abstractness, distance_from_main_sequence).

## Setup Workflow

1. Run ` + "`archlint collect <project-dir> -o architecture.yaml`" + ` to see what components exist.
2. Create ` + "`.archlint.yaml`" + ` with rules appropriate for the project (see profiles above).
3. Customize thresholds: inspect the architecture.yaml output to understand component counts and structure, then set thresholds that flag real issues without false positives.
4. Add exclude patterns for components that legitimately break rules (e.g., cmd/* packages often have high fan-out).
5. Customize layer_violations.params.layers to match the project's actual package layout.
6. Run ` + "`archlint validate architecture.yaml`" + ` to check the architecture.

## Questions to Ask the User

Before generating .archlint.yaml, gather this information:

1. **Project maturity**: Is this a new project, active development, or mature/maintenance? (Determines how strict to be.)
2. **Architecture style**: Does the project follow clean architecture, hexagonal, layered, or no specific pattern? (Determines which architecture style rules to enable.)
3. **Package layout**: What are the top-level packages and their roles? (Needed to configure layer_violations and forbidden_dependencies correctly.)
4. **Known exceptions**: Are there components that intentionally break conventions? (e.g., a god-object orchestrator, a high-fan-out main package.) These should be added to exclude lists.
5. **Error vs warning preference**: Should violations block CI (error_on_violation: true) or just warn? (Depends on whether this is aspirational or enforced.)
6. **Focus areas**: Any specific concerns? (e.g., "we have circular dependencies we want to break", "we want to enforce domain isolation".) Helps prioritize which rules to enable and tune.
`
