package utility

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

func DiffMaps[T any](current, new map[string]T) Changes[T] {
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
