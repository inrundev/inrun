//go:build !runtime && !gateway

package cli

// E2E status strings for registry info / list output.

func e2eVerified(suffix string) string {
	s := green("✓ Verified")
	if suffix != "" {
		s += " · " + suffix
	}
	return s
}

func e2eSkipped() string {
	return yellow("⊘ Skipped") + " (pushed with --force or --no-e2e)"
}

func skippedShort() string {
	return yellow("⊘ Skipped")
}

func e2eNotVerified() string {
	return gray("- Not verified")
}

func simulateVerified(suffix string) string {
	s := green("✓ Verified")
	if suffix != "" {
		s += " · " + suffix
	}
	return s
}

func simulateSkipped() string {
	return yellow("⊘ Skipped") + " (pushed with --force or --no-simulate)"
}

func simulateNoAssertion() string {
	return yellow("⚠ No assertions") + " (add expect: to simulate.yaml to enforce behavior)"
}

func noAssertion() string {
	return yellow("⚠ No assertions")
}

// Diff change icons used in simulate output.

func iconAdded() string   { return green("+") }
func iconRemoved() string { return red("-") }
func iconChanged() string { return yellow("~") }
