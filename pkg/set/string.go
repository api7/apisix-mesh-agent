package set

import "sort"

// StringSet represents a set which elements are string.
type StringSet map[string]struct{}

// Add adds an element to set.
func (set StringSet) Add(e string) {
	set[e] = struct{}{}
}

// Equal compares two string set and checks whether they are identical.
func (set StringSet) Equal(set2 StringSet) bool {
	if len(set) != len(set2) {
		return false
	}
	for e := range set2 {
		if _, ok := set[e]; !ok {
			return false
		}
	}
	for e := range set {
		if _, ok := set2[e]; !ok {
			return false
		}
	}
	return true
}

// Strings converts the string set to a string slice.
func (set StringSet) Strings() []string {
	s := make([]string, 0, len(set))
	for e := range set {
		s = append(s, e)
	}
	return s
}

// OrderedStrings converts the string set to a sorted string slice.
func (set StringSet) OrderedStrings() []string {
	s := set.Strings()
	sort.Strings(s)
	return s
}
