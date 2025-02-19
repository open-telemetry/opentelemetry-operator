// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diff

// Changes is the result of the difference between two maps – items that are added and items that are removed
// This map is used to reconcile state differences.
type Changes[T Hasher] struct {
	additions map[string]T
	removals  map[string]T
}

type Hasher interface {
	Hash() string
}

func NewChanges[T Hasher](additions map[string]T, removals map[string]T) Changes[T] {
	return Changes[T]{additions: additions, removals: removals}
}

func (c Changes[T]) Additions() map[string]T {
	return c.additions
}

func (c Changes[T]) Removals() map[string]T {
	return c.removals
}

// Maps generates Changes for two maps with the same type signature by checking for any removals and then checking for
// additions.
// TODO: This doesn't need to create maps, it can return slices only. This function doesn't need to insert the values.
func Maps[T Hasher](current, new map[string]T) Changes[T] {
	additions := map[string]T{}
	removals := map[string]T{}
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
	return Changes[T]{
		additions: additions,
		removals:  removals,
	}
}
