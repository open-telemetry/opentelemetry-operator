// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diff

// Changes is the result of the difference between two maps – items that are added and items that are removed
// This map is used to reconcile state differences.
type Changes[K comparable, T Hasher[K]] struct {
	additions map[K]T
	removals  map[K]T
}

type Hasher[K comparable] interface {
	Hash() K
}

func NewChanges[K comparable, T Hasher[K]](additions map[K]T, removals map[K]T) Changes[K, T] {
	return Changes[K, T]{additions: additions, removals: removals}
}

func (c Changes[K, T]) Additions() map[K]T {
	return c.additions
}

func (c Changes[K, T]) Removals() map[K]T {
	return c.removals
}

// Maps generates Changes for two maps with the same type signature by checking for any removals and then checking for
// additions.
// TODO: This doesn't need to create maps, it can return slices only. This function doesn't need to insert the values.
func Maps[K comparable, T Hasher[K]](current, new map[K]T) Changes[K, T] {
	additions := map[K]T{}
	removals := map[K]T{}
	for key, newValue := range new {
		if currentValue, found := current[key]; !found {
			additions[key] = newValue
		} else if currentValue.Hash() != newValue.Hash() {
			additions[key] = newValue
			removals[key] = currentValue
		}
	}
	for key, value := range current {
		if _, found := new[key]; !found {
			removals[key] = value
		}
	}
	return Changes[K, T]{
		additions: additions,
		removals:  removals,
	}
}
