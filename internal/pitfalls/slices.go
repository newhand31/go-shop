package pitfalls

// firstBuggy returns the first n items of a cart and then appends one more. A slice is a window (pointer, length,
// capacity) onto an array, and cart[:n] is a window onto the SAME array with spare capacity, so append writes into it.
func firstBuggy(cart []int, n, extra int) []int {
	head := cart[:n]
	return append(head, extra)
}

// firstFixed caps the window with a third index (cart[:n:n]): length n, capacity n. The append finds no room, allocates
// a new array and copies, so the caller's cart is untouched.
func firstFixed(cart []int, n, extra int) []int {
	head := cart[:n:n]
	return append(head, extra)
}
