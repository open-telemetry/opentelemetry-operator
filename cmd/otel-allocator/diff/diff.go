package diff

// Changes is the result of the difference between two maps – items that are added and items that are removed
// This map is used to reconcile state differences.
type Changes[T any] struct {
	additions map[string]T
	removals  map[string]T
}

func (c Changes[T]) Additions() map[string]T {
	return c.additions
}

func (c Changes[T]) Removals() map[string]T {
	return c.removals
}

// Maps generates Changes for two maps with the same type signature by checking for any removals and then checking for
// additions.
func Maps[T any](current, new map[string]T) Changes[T] {
	additions := map[string]T{}
	removals := map[string]T{}
	// Used as a set to check for removed items
	newMembership := map[string]bool{}
	for key, value := range new {
		if _, found := current[key]; !found {
			additions[key] = value
		}
		newMembership[key] = true
	}
	for key, value := range current {
		if _, found := newMembership[key]; !found {
			removals[key] = value
		}
	}
	return Changes[T]{
		additions: additions,
		removals:  removals,
	}
}
