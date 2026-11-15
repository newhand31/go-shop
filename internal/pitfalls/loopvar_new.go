package pitfalls

// counterNew collects closures that each return the loop variable. This file follows the module's language version
// (go.mod, Go 1.22 or later), where every iteration gets its own copy of i.
func counterNew(n int) []func() int {
	var fns []func() int
	for i := 0; i < n; i++ {
		fns = append(fns, func() int { return i })
	}
	return fns
}
