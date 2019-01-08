package lib

// Allocator is responsible to determine which nodes ID should fail in a given
// experiment amongst all nodes. One allocator could set space out regularly
// failing node, or put a maximum of failing node in one region of the ID space,
// etc.
type Allocator interface {
	// Allocate returns the list of IDs of node that must be online during the
	// experiements. IDs not returned in the slice should not be setup.
	Allocate(total, offline int) []int
}

type linearAllocator struct{}

func (l *linearAllocator) Allocate(total, offline int) []int {
	var bucket = total + 1
	if offline != 0 {
		bucket, _ = Divmod(total, offline)
	}
	ids := make([]int, 0, total-offline)
	for i := 0; i < total; i++ {
		if offline > 0 && i%bucket == 0 {
			// remove the node of the bucket
			offline--
			continue
		}
		ids = append(ids, i)
	}
	// take out the end
	if offline > 0 {
		ids = ids[:len(ids)-offline]
	}
	return ids
}
