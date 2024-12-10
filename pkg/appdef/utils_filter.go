/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Returns all types that match the filter.
func FilterMatches(f IFilter, types SeqType) SeqType {
	return func(yield func(IType) bool) {
		for t := range types {
			if f.Match(t) {
				if !yield(t) {
					return
				}
			}
		}
	}
}

// Returns the first type that matches the filter.
// Returns nil if no types match the filter.
func FirstFilterMatch(f IFilter, types SeqType) IType {
	for t := range types {
		if f.Match(t) {
			return t
		}
	}
	return nil
}
