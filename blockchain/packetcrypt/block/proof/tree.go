// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/bits"
	"strings"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/blockchain/packetcrypt/pcutil"
)

// How does it work:
// the tree is basically a Merkle tree with ranges in order to prevent duplicate entries.
// The problem which this tree solves is as follows: Someone could duplicate an entry in
// the tree a number of times and as long as their dupe doesn't come up more than once in
// the proof, the proof is valid. In order to prevent that, we require the the prover to do
// the following:
//
// 1. Sort the set of announcements by hash
// 2. Drop any announcements for which the first 8 bytes of the hash are all duplicate,
//    all zero, or all ff
// 3. Insert a fake entry at the beginning which is all zero, this will become important later
// 4. If the number of entries is odd, insert a fake entry at the end which is all ff
// 5. Next to each announcement hash, place 2 8 byte values: "start" (the last 8 bytes of the
//    announcement hash) and "end" the last 8 bytes of the next announcement hash.
// 6. Hash each pair of ann hashes together, hashing over the ann hash as well as the start and
//    end tags.
// 7. Copy the start number from the left side and the end number from the right side, up to
//    the newly created merkle node.
// 8. If the number of merkle nodes in this layer is odd, insert a fake node whose hash is all
//    ff and whose start is ff and end is ff
// 9. Go back to 6 and repeat for each layer until there is only 1 merkle node left
// 10. Check that the start number is 0 and the end number is all ff and output the hash.
//
// In order to make such a hash tree with any duplicates, there must be overlapping ranges.
// Because the computation of the root requires the validator to know the ranges of all
// subtrees of the tree containing each announcement, he can be confident that none of the
// announcements which were provided in the proof have a valid duplicate. Invalid duplicates
// are the same as normal invalid announcements and the security properties of PacketCrypt
// vs. invalid/fake announcements is discussed elsewhere.
//
// In order to provide the validator with enough information for him to deduce the ranges and
// thus recompute the tree root, we need to provide at least the difference between the "start"
// and the "end" with each provided merkle node with one exception:
// Left-side leaf nodes do not need to contain a range, because the node itself is an announcement
// hash and the "start" of the node is known by looking at the hash.
//
// If we were proving the existence of just one announcement in the tree, it would be this easy.
// However, because we are proving 4, there are also a number of entries which are COMPUTABLE,
// meaning that both of their children are known and so we can compute them completely.

type TreeNode struct {
	tbl        *Tree
	childLeft  int
	childRight int

	// if parent is -1 then this is the root entry
	parent int
	flags  Flag
	raNge  uint64
	start  uint64
	end    uint64
	hash   [32]byte
}

// Flags gets the flags for this node
func (te *TreeNode) Flags() Flag {
	return te.flags
}

// Hash gets the memory location of the entry hash (mutable)
func (te *TreeNode) Hash() []byte {
	return te.hash[:]
}

// Start gets the uint64 representing the start of the node's range
func (te *TreeNode) Start() uint64 {
	return te.start
}

// End gets the end of the node's range (next entry hash - 1)
func (te *TreeNode) End() uint64 {
	return te.end
}

// Range gets the difference between the start and the end
// if it is not yet known then the result will be 0
func (te *TreeNode) Range() uint64 {
	return te.raNge
}

// Number gives the id of this node in the tree
func (te *TreeNode) Number() int {
	p := te.GetParent()
	if p == nil {
		return 0
	}
	if te.flags&FRight == FRight {
		return p.childRight
	}
	return p.childLeft
}

func (te *TreeNode) recompute() bool {
	if te.flags.has(FHasStart | FHasEnd | FHasRange) {
		// We have start, end, and range
		if !te.flags.has(FHasStart | FHasEnd | FHasRange | FHasHash) {
			// waiting on the hash
			return true
		}
		sib := te.GetSibling()
		if sib == nil {
			// we hit the root
			return true
		}
		if !sib.flags.has(FHasStart | FHasEnd | FHasRange | FHasHash) {
			// sibling doesn't have everything yet
			return true
		}
		// We have everything, lets compute our parent's hash
		p := te.GetParent()
		a := te
		b := sib
		if te.flags.has(FRight) {
			a = sib
			b = te
		}
		var buf [96]byte
		copy(buf[:32], a.Hash())
		binary.LittleEndian.PutUint64(buf[32:40], a.Start())
		binary.LittleEndian.PutUint64(buf[40:48], a.End())
		copy(buf[48:80], b.Hash())
		binary.LittleEndian.PutUint64(buf[80:88], b.Start())
		binary.LittleEndian.PutUint64(buf[88:96], b.End())
		var phash [32]byte
		pcutil.HashCompress(phash[:], buf[:])
		if !p.SetHash(phash[:]) {
			return false
		}
		if !p.SetStart(a.Start()) {
			return false
		}
		if !p.SetEnd(b.End()) {
			return false
		}
		return true
	} else if te.flags.has(FHasStart | FHasEnd) {
		// no range
		if te.flags.has(FHasStart | FHasEnd) {
			// got both start and end, infer the range
			return te.SetRange(te.End() - te.Start())
		}
		// it's fine, we have only start or only end
		return true
	} else if te.flags.has(FHasEnd | FHasRange) {
		// got the end, missing the start
		return te.SetStart(te.End() - te.Range())
	} else if te.flags.has(FHasStart | FHasRange) {
		// got the start, missing the end
		return te.SetEnd(te.Range() + te.Start())
	} else {
		// just has one of start, end, or range.. nothing to do right now
		return true
	}
}

// SetHash sets the provided byte array to the entry hash and flags as hash present
func (te *TreeNode) SetHash(h []byte) bool {
	log.Tracef("TreeNode[%d].SetHash(%s)", te.Number(), hex.EncodeToString(h))
	if te.flags.has(FHasHash) {
		return bytes.Equal(h, te.hash[:])
	}
	copy(te.hash[:], h)
	te.flags |= FHasHash
	if te.flags.has(FLeaf) {
		// If it's a leaf (announcement) then the hash bytes *are* the start
		// SetStart triggers recompute()
		return te.SetStart(binary.LittleEndian.Uint64(h[:8]))
	}
	return te.recompute()
}

// SetStart sets the start point of of the node's range
func (te *TreeNode) SetStart(s uint64) bool {
	log.Tracef("TreeNode[%d].SetStart(%016x)", te.Number(), s)
	if te.flags.has(FHasStart) {
		return s == te.Start()
	}
	te.start = s
	te.flags |= FHasStart
	if te.flags.has(FRight) {
		sib := te.GetSibling()
		if !sib.SetEnd(s) {
			return false
		}
	}
	return te.recompute()
}

// SetEnd sets the end of the node's range, if the range is
// known then the start will be inferred and saved.
func (te *TreeNode) SetEnd(s uint64) bool {
	log.Tracef("TreeNode[%d].SetEnd(%016x)", te.Number(), s)
	if te.flags.has(FHasEnd) {
		return s == te.End()
	}
	te.end = s
	te.flags |= FHasEnd
	if !te.flags.has(FRight) {
		sib := te.GetSibling()
		// Skip if we're the root
		if sib != nil && !sib.SetStart(s) {
			return false
		}
	}
	return te.recompute()
}

// SetRange sets the range for a node, if the start is known
// then the start and end are computed and saved.
// If the range has already been set then this function returns false
func (te *TreeNode) SetRange(s uint64) bool {
	log.Tracef("TreeNode[%d].SetEnd(%016x)", te.Number(), s)
	if te.flags.has(FHasRange) {
		return s == te.Range()
	}
	te.raNge = s
	te.flags |= FHasRange
	return te.recompute()
}

// GetParent returns the node's parent, nil for the root node
func (te *TreeNode) GetParent() *TreeNode {
	if te == &te.tbl.entries[0] {
		return nil
	}
	return &te.tbl.entries[te.parent]
}

// GetSibling returns the node's sibling, nil for the root node
func (te *TreeNode) GetSibling() *TreeNode {
	p := te.GetParent()
	if p == nil {
		return nil
	}
	if (te.flags & FRight) != 0 {
		return &te.tbl.entries[p.childLeft]
	}
	return &te.tbl.entries[p.childRight]
}

// HasExplicitRange returns true if the entry requires an explicit range in order to validate
func (te *TreeNode) HasExplicitRange() bool {
	// We need the start and end of any leaf node in order to compute, however a left leaf
	// we can infer the end by the hash of the announcement itself. However a right leaf
	// range goes to the next (unknown) entry so we need to provide the range explicitly.
	// If course, if it's a pad entry, we don't need to provide it.
	if (te.flags & (FLeaf | FRight | FPadEntry)) == (FLeaf | FRight) {
		return true
	}

	// Anything remaining that's:
	// * Not a LEAF - if it's a leaf node then it's left and can be inferred
	// * Not COMPUTABLE - if it's computable then we know both it's children, can infer everything
	// * Not a PAD_ENTRY - pad entries have a hardcoded 0 range (ffff - ffff)
	// * Not a PAD_SIBLING - because we know the start of the next entry (it's hardcoded)
	//       in practice all PAD_SIBLINGS should be COMPUTABLE but we add this for completeness.
	return (te.flags & (FLeaf | FComputable | FPadEntry | FPadSibling)) == 0
}

// returns a list of bits representing the left and right turns which one makes in the tree
// to find the announcement entry, based on the announcement number
func pathForNum(num uint64, branchHeight int) uint64 {
	return bits.Reverse64(num) >> uint(64-branchHeight)
}

type Tree struct {
	branchHeight int
	entries      []TreeNode
}

// GetRoot returns the entry which is at the root of the tree, this one's hash is committed
// in the coinbase.
func (t *Tree) GetRoot() *TreeNode { return &t.entries[0] }

// GetAnnEntry gets the table netry corresponding to a particular announcement.
func (t *Tree) GetAnnEntry(annNum uint64) *TreeNode {
	path := pathForNum(annNum, t.branchHeight)
	e := t.GetRoot()
	for i := 0; i < t.branchHeight; i++ {
		next := e.childLeft
		if (path & 1) == 1 {
			next = e.childRight
		}
		e = &t.entries[next]
		path >>= 1
	}
	if !e.flags.has(FLeaf | FComputable) {
		panic("ann entry is not a computable leaf")
	}
	return e
}

func dumpNode(n *TreeNode, padding int) {
	pad := strings.Repeat("  ", padding)
	log.Tracef("%s%d %016x - %016x %v", pad, n.Number(), n.start, n.end, n.flags)
	if n.childLeft > -1 {
		dumpNode(&n.tbl.entries[n.childLeft], padding+1)
	}
	if n.childRight > -1 {
		dumpNode(&n.tbl.entries[n.childRight], padding+1)
	}
}

func (t *Tree) DumpTree() {
	dumpNode(t.GetRoot(), 0)
}

func mkEntries(
	tree *Tree,
	annIdxs *[4]uint64,
	bits uint64,
	iDepth uint,
	parentNum int,
	annCount uint64,
) er.R {
	tree.entries = tree.entries[:len(tree.entries)+1]
	eNum := len(tree.entries) - 1
	e := &tree.entries[eNum]
	e.parent = parentNum
	e.tbl = tree

	mask := uint64(0xffffffffffffffff) << iDepth

	flags := Flag(0)
	if ((bits >> iDepth) & 1) == 1 {
		flags |= FRight
	}
	if iDepth == 0 {
		flags |= FLeaf
	}
	if (bits & mask) == 0 {
		flags |= FFirstEntry
		flags |= FHasStart
	}

	for i := 0; i < 4; i++ {
		if ((annIdxs[i] ^ bits) & mask) != 0 {
			continue
		}
		e.flags = flags | FComputable
		if (e.flags & FLeaf) == FLeaf {
			if bits != annIdxs[i] {
				panic("computable leaf entry which is not an announcement, should not happen")
			}
			// this is the announcement
			e.childLeft = -1
			e.childRight = -1
			return nil
		}
		e.childLeft = len(tree.entries)
		if err := mkEntries(tree, annIdxs, bits, iDepth-1, eNum, annCount); err != nil {
			return err
		}
		e.childRight = len(tree.entries)
		nextBits := bits | (uint64(1) << (iDepth - 1))
		if err := mkEntries(tree, annIdxs, nextBits, iDepth-1, eNum, annCount); err != nil {
			return err
		}

		if tree.entries[e.childRight].flags.has(FPadEntry) {
			if !tree.entries[e.childLeft].flags.has(FComputable) {
				return er.New("pad sibling which is not computable")
			}
			tree.entries[e.childLeft].flags |= FPadSibling
		}
		return nil
	}

	// Not a parent of any announcement

	// So it's a provided entry and so will not have children
	e.childLeft = -1
	e.childRight = -1

	if bits >= annCount {
		// it's a pad entry
		if e.flags.has(FRight) {
			return er.New("right-side pad entry, nonsense")
		}
		e.flags = flags | FPadEntry | FHasHash | FHasRange | FHasStart | FHasEnd
		pcutil.Memset(e.Hash(), 0xff)
		e.start = 0xffffffffffffffff
		e.end = 0xffffffffffffffff
		return nil
	}

	// It's a sibling for which data must be provided
	e.flags = flags
	return nil
}

func NewTree(annCount uint64, annIdxs *[4]uint64) (*Tree, er.R) {
	// sanity check
	for i := 0; i < 4; i++ {
		if annIdxs[i] >= annCount {
			return nil, er.New("invalid index ids or annCount")
		}
	}

	branchHeight := pcutil.Log2ceil(annCount)
	capacity := branchHeight * 4 * 3
	out := new(Tree)
	out.entries = make([]TreeNode, 0, capacity)
	out.branchHeight = branchHeight
	if err := mkEntries(out, annIdxs, 0, uint(branchHeight), -1, annCount); err != nil {
		return nil, er.Errorf("mkEntries returned an error, this is a bug %v", err)
	}
	return out, nil
}
