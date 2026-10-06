package e2e

import (
	"fmt"
	"strings"
	"time"
)

// Markdown renders the result as a GitHub-flavoured markdown summary block.
func (r *Result) Markdown() string {
	var passed, failed, skipped []CaseResult
	for _, c := range r.Cases {
		switch {
		case c.Skipped:
			skipped = append(skipped, c)
		case c.Passed:
			passed = append(passed, c)
		default:
			failed = append(failed, c)
		}
	}

	var b strings.Builder
	b.WriteString("## E2E Results: " + r.Name + "\n\n")

	if len(passed) > 0 {
		b.WriteString("| Passed | Test | Time |\n")
		b.WriteString("|---|---|---|\n")
		for _, c := range passed {
			b.WriteString(fmt.Sprintf("| ✅ | %s | %s |\n", c.Name, c.Elapsed.Round(time.Millisecond)))
		}
		b.WriteString("\n")
	}

	if len(failed) > 0 {
		b.WriteString("| Failed | Test | Time | Error |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, c := range failed {
			msg := ""
			if c.Err != nil {
				msg = strings.ReplaceAll(strings.TrimSpace(c.Err.Error()), "\n", ". ")
			}
			b.WriteString(fmt.Sprintf("| ❌ | %s | %s | %s |\n", c.Name, c.Elapsed.Round(time.Millisecond), msg))
		}
		b.WriteString("\n")
	}

	if len(skipped) > 0 {
		b.WriteString("| Skipped | Test |\n")
		b.WriteString("|---|---|\n")
		for _, c := range skipped {
			b.WriteString(fmt.Sprintf("| ⏭ | %s |\n", c.Name))
		}
		b.WriteString("\n")
	}

	b.WriteString("**" + r.Summary() + "**\n")
	return b.String()
}

// Result holds the outcome of a complete E2E run.
// It is returned by Run and consumed by inrun push to embed
// verification metadata into OCI annotations.
type Result struct {
	Name    string
	Cases   []CaseResult
	Elapsed time.Duration
}

// CaseResult holds the outcome of a single expectation.
type CaseResult struct {
	Name    string
	Passed  bool
	Skipped bool
	Elapsed time.Duration
	Err     error
}

// AllPassed returns true when every non-skipped expectation passed.
func (r *Result) AllPassed() bool {
	for _, c := range r.Cases {
		if !c.Passed && !c.Skipped {
			return false
		}
	}
	return true
}

// Passed returns the number of passing expectations.
func (r *Result) Passed() int {
	n := 0
	for _, c := range r.Cases {
		if c.Passed {
			n++
		}
	}
	return n
}

// Skipped returns the number of skipped expectations.
func (r *Result) Skipped() int {
	n := 0
	for _, c := range r.Cases {
		if c.Skipped {
			n++
		}
	}
	return n
}

// Failed returns the number of failed expectations.
func (r *Result) Failed() int {
	n := 0
	for _, c := range r.Cases {
		if !c.Passed && !c.Skipped {
			n++
		}
	}
	return n
}

// Total returns the total number of expectations.
func (r *Result) Total() int { return len(r.Cases) }

// Duration returns the total elapsed time as a human-readable string (e.g. "45s").
func (r *Result) Duration() string {
	return r.Elapsed.Round(time.Second).String()
}

// Summary returns a one-line result string, e.g. "3 passed, 0 failed, 1 skipped (45s)".
func (r *Result) Summary() string {
	total := len(r.Cases)
	s := fmt.Sprintf("%d of %d passed", r.Passed(), total)
	if n := r.Skipped(); n > 0 {
		s += fmt.Sprintf(", %d skipped", n)
	}
	return s + fmt.Sprintf(" (%s)", r.Duration())
}

// ImportResult holds the outcome of one imported E2E file in a suite run.
type ImportResult struct {
	// Path is the import path as declared in imports: (e.g. "./01-basic/e2e.yaml").
	Path string
	// Result is the structured outcome. Nil when the import failed to load or
	// the runner itself errored before producing any expectations.
	Result *Result
	// Err is non-nil when the import failed (load error, Inrun not ready,
	// or one or more expectations failed).
	Err error
}

// Name returns the display name — the E2E metadata.name when available,
// otherwise the import path.
func (i ImportResult) Name() string {
	if i.Result != nil && i.Result.Name != "" {
		return i.Result.Name
	}
	return i.Path
}

// Passed reports whether this import succeeded.
func (i ImportResult) Passed() bool { return i.Err == nil }

// printImportSummary prints the suite results table and returns a non-nil
// error when any import failed.
func printImportSummary(suiteName string, results []ImportResult) error {
	passed := 0
	for _, ir := range results {
		if ir.Passed() {
			passed++
		}
	}

	var total time.Duration
	for _, ir := range results {
		if ir.Result != nil {
			total += ir.Result.Elapsed
		}
	}

	fmt.Printf("\n─── Suite Results: %s ───\n\n", suiteName)
	for _, ir := range results {
		icon := "✓"
		if !ir.Passed() {
			icon = "✗"
		}

		score := "—"
		elapsed := ""
		if ir.Result != nil {
			score = fmt.Sprintf("%d / %d", ir.Result.Passed(), ir.Result.Total())
			elapsed = ir.Result.Duration()
		}

		errSuffix := ""
		if ir.Err != nil {
			msg := ir.Err.Error()
			// Trim the "import ./path: " prefix if present so the table stays narrow.
			if idx := strings.Index(msg, ": "); idx >= 0 {
				msg = msg[idx+2:]
			}
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			errSuffix = "  → " + msg
		}

		fmt.Printf("  %s  %-44s  %-6s  %-6s%s\n", icon, ir.Name(), score, elapsed, errSuffix)
	}

	elapsed := total.Round(time.Second).String()
	fmt.Printf("\n  %d of %d imports passed  (%s)\n", passed, len(results), elapsed)

	if passed < len(results) {
		return fmt.Errorf("%d of %d imports failed", len(results)-passed, len(results))
	}
	return nil
}
