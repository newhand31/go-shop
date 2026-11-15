package pitfalls

import (
	"errors"
	"slices"
	"testing"
)

func TestNilPointerInsideAnErrorIsNotNil(t *testing.T) {
	if err := findBuggy(1); err == nil {
		t.Fatal("the trap did not trigger: want a non-nil interface holding a nil pointer")
	} else if typed, ok := err.(*NotFoundError); !ok || typed != nil {
		t.Fatalf("err = %#v, want a nil *NotFoundError inside a non-nil error", err)
	}
	if err := findFixed(1); err != nil {
		t.Fatalf("findFixed(1) = %v, want a nil interface", err)
	}
}

func TestRealErrorsStillWork(t *testing.T) {
	for name, find := range map[string]func(int) error{"buggy": findBuggy, "fixed": findFixed} {
		var nf *NotFoundError
		if err := find(7); !errors.As(err, &nf) || nf.ProductID != 7 {
			t.Errorf("%s: find(7) = %v, want *NotFoundError for product 7", name, err)
		}
	}
}

func results(fns []func() int) []int {
	var out []int
	for _, f := range fns {
		out = append(out, f())
	}
	return out
}

func TestEachIterationHasItsOwnLoopVariableSinceGo122(t *testing.T) {
	if got, want := results(counterNew(3)), []int{0, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("counterNew = %v, want %v", got, want)
	}
}

func TestOneSharedLoopVariableBeforeGo122(t *testing.T) {
	if got, want := results(counterOld(3)), []int{3, 3, 3}; !slices.Equal(got, want) {
		t.Errorf("counterOld = %v, want %v (every closure sees the final i)", got, want)
	}
}

func TestAppendOnASubSliceOverwritesTheOriginal(t *testing.T) {
	cart := []int{10, 20, 30, 40}
	got := firstBuggy(cart, 2, 99)
	if want := []int{10, 20, 99}; !slices.Equal(got, want) {
		t.Errorf("result = %v, want %v", got, want)
	}
	if want := []int{10, 20, 99, 40}; !slices.Equal(cart, want) {
		t.Errorf("cart = %v, want %v: the third item was overwritten", cart, want)
	}
}

func TestCappingTheCapacityKeepsTheOriginalSafe(t *testing.T) {
	cart := []int{10, 20, 30, 40}
	got := firstFixed(cart, 2, 99)
	if want := []int{10, 20, 99}; !slices.Equal(got, want) {
		t.Errorf("result = %v, want %v", got, want)
	}
	if want := []int{10, 20, 30, 40}; !slices.Equal(cart, want) {
		t.Errorf("cart = %v, want it unchanged", cart)
	}
}
