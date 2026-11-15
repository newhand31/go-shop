//go:build go1.21

package pitfalls

// counterOld is the same code as counterNew, but the build line above pins this one file to Go 1.21 rules (a go:build
// line may lower the language version of a single file). Before Go 1.22 the loop had ONE i for all iterations, so every
// closure sees the value the loop ended with. This file exists to show the old behaviour; do not write new code like this.
func counterOld(n int) []func() int {
	var fns []func() int
	for i := 0; i < n; i++ {
		fns = append(fns, func() int { return i })
	}
	return fns
}
