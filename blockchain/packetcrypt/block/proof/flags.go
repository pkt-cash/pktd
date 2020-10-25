// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

// Go is dumb, you have to write everything manually

// Flag is a flag which is applied to tree entries to indicate their state
type Flag uint16

const (
	// Constants: These will not be changed after being set by tree.go

	// FComputable means no information will be provided about this node because
	// it can be fully computed (both child nodes are known)
	FComputable Flag = 1

	// FPadEntry means this is a pad entry (added to make the number of nodes even)
	// no information is provided because it's hard-wired to all ff
	FPadEntry Flag = (1 << 1)

	// FLeaf this node is canadian, it corresponds to an announcement hash
	// (including the ones which we're proving)
	FLeaf Flag = (1 << 2)

	// FRight means this node is a right-hand node, if this flag is not present then
	// the node is left-hand or is the root
	FRight Flag = (1 << 3)

	// FPadSibling left sibling of a pad entry, in practice this will always be
	// COMPUTABLE and so no information is provided anyway but it is here for completeness.
	FPadSibling Flag = (1 << 4)

	// FFirstEntry means the node is the first node in the layer (index 0), this
	// means the "start" for this node is known, it is 0
	FFirstEntry Flag = (1 << 5)

	// The following are mutated by pcp.go

	// FHasHash is set when we know the hash for a given node, it means the hash
	// value is meaningful
	FHasHash Flag = (1 << 8)

	// FHasRange is set when we know the range for a node
	FHasRange Flag = (1 << 9)

	// FHasStart is set when we know the start for a node.
	FHasStart Flag = (1 << 10)

	// FHasEnd means that we know the end of a node's range.
	FHasEnd Flag = (1 << 11)
)

func (f Flag) String() string {
	out := ""
	if f&FComputable != 0 {
		out += "|FComputable"
	}
	if f&FPadEntry != 0 {
		out += "|FPadEntry"
	}
	if f&FLeaf != 0 {
		out += "|FLeaf"
	}
	if f&FRight != 0 {
		out += "|FRight"
	}
	if f&FPadSibling != 0 {
		out += "|FPadSibling"
	}
	if f&FFirstEntry != 0 {
		out += "|FFirstEntry"
	}
	if f&FHasHash != 0 {
		out += "|FHasHash"
	}
	if f&FHasRange != 0 {
		out += "|FHasRange"
	}
	if f&FHasStart != 0 {
		out += "|FHasStart"
	}
	if f&FHasEnd != 0 {
		out += "|FHasEnd"
	}
	return out[1:]
}

func (f Flag) has(ff Flag) bool {
	return f&ff == ff
}
