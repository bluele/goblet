package goblet

// Refs is a reference slice
type Refs []interface{}

// Len returns length for Refs
func (rf Refs) Len() int {
	length := 0
	for _, ref := range rf {
		switch ref.(type) {
		case string:
			length++
		case *ParallelReference:
			length += ref.(*ParallelReference).Len()
		}
	}
	return length
}

// ParallelReference is a reference slice for allowing resolve dependencies concurrently
type ParallelReference struct {
	refs []string
}

// ParallelRefs returns a ParallelReference object.
func ParallelRefs(refs ...string) *ParallelReference {
	return &ParallelReference{refs: refs}
}

// Len returns a length of reference slice.
func (rf *ParallelReference) Len() int {
	return len(rf.refs)
}
