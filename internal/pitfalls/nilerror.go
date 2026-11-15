package pitfalls

import "fmt"

// NotFoundError says which product was missing.
type NotFoundError struct{ ProductID int }

func (e *NotFoundError) Error() string { return fmt.Sprintf("product %d not found", e.ProductID) }

// findBuggy returns an error interface that holds a nil *NotFoundError. An interface value is a pair (type, pointer);
// this one is (*NotFoundError, nil), and a pair with a type is not the nil interface. The caller's `err != nil` is true.
func findBuggy(id int) error {
	var err *NotFoundError // the typed nil: its type is known, its pointer is empty
	if id != 1 {
		err = &NotFoundError{ProductID: id}
	}
	return err
}

// findFixed returns the literal nil when there is nothing to report, so the interface is empty: (no type, no pointer).
func findFixed(id int) error {
	if id != 1 {
		return &NotFoundError{ProductID: id}
	}
	return nil
}
