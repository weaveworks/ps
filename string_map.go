package ps

import (
	"bytes"
	"fmt"
)

// String maps using explicit values instead of interfaces to
// avoid unnecessary garbage/castings.

type StringMap struct {
	count    int
	hash     uint64 // hash of the key (used for tree balancing)
	key      string
	value    string
	children [childCount]*StringMap
}

var nilStringMap = &StringMap{}

// Recursively set nilMap's subtrees to point at itself.
// This eliminates all nil pointers in the map structure.
// All map nodes are created by cloning this structure so
// they avoid the problem too.
func init() {
	for i := range nilMap.children {
		nilMap.children[i] = nilMap
	}
}

// NewMap allocates a new, persistent map from strings to values of
// any type.
// This is currently implemented as a path-copying binary tree.
func NewStringMap() *StringMap {
	return nilMap
}

func (self *StringMap) IsNil() bool {
	return self == nilMap
}

// clone returns an exact duplicate of a tree node
func (self *StringMap) clone() *StringMap {
	var m StringMap
	m = *self
	return &m
}

// Set returns a new map similar to this one but with key and value
// associated.  If the key didn't exist, it's created; otherwise, the
// associated value is changed.
func (self *StringMap) Set(key string, value string) Map {
	hash := hashKey(key)
	return self.setLowLevel(self, hash, hash, key, value)
}

func (self *StringMap) setLowLevel(partialHash, hash uint64, key string, value string) *StringMap {
	if self.IsNil() { // an empty tree is easy
		m := self.clone()
		m.count = 1
		m.hash = hash
		m.key = key
		m.value = value
		return m
	}

	if hash != self.hash {
		m := self.clone()
		i := partialHash % childCount
		m.children[i] = setLowLevel(self.children[i], partialHash>>shiftSize, hash, key, value)
		recalculateCount(m)
		return m
	}

	// did we find a hash collision?
	if key != self.key {
		oops := fmt.Sprintf("Hash collision between: '%s' and '%s'.  Please report to https://github.com/mndrix/ps/issues/new", self.key, key)
		panic(oops)
	}

	// replacing a key's previous value
	m := self.clone()
	m.value = value
	return m
}

// modifies a map by recalculating its key count based on the counts
// of its subtrees
func (m *StringMap) recalculateCount() {
	count := 0
	for _, t := range m.children {
		count += t.Size()
	}
	m.count = count + 1 // add one to count ourself
}

func (m *StringMap) Delete(key string) Map {
	hash := hashKey(key)
	newMap, _ := deleteLowLevel(m, hash, hash)
	return newMap
}

func deleteLowLevel(self *StringMap, partialHash, hash uint64) (*StringMap, bool) {
	// empty trees are easy
	if self.IsNil() {
		return self, false
	}

	if hash != self.hash {
		i := partialHash % childCount
		child, found := deleteLowLevel(self.children[i], partialHash>>shiftSize, hash)
		if !found {
			return self, false
		}
		newMap := self.clone()
		newMap.children[i] = child
		newMap.recalculateCount()
		return newMap, true // ? this wasn't in the original code
	}

	// we must delete our own node
	if self.isLeaf() { // we have no children
		return nilMap, true
	}
	/*
	   if self.subtreeCount() == 1 { // only one subtree
	       for _, t := range self.children {
	           if t != nilMap {
	               return t, true
	           }
	       }
	       panic("Tree with 1 subtree actually had no subtrees")
	   }
	*/

	// find a node to replace us
	i := -1
	size := -1
	for j, t := range self.children {
		if t.Size() > size {
			i = j
			size = t.Size()
		}
	}

	// make chosen leaf smaller
	replacement, child := self.children[i].deleteLeftmost()
	newMap := replacement.clone()
	for j := range self.children {
		if j == i {
			newMap.children[j] = child
		} else {
			newMap.children[j] = self.children[j]
		}
	}
	recalculateCount(newMap)
	return newMap, true
}

// delete the leftmost node in a tree returning the node that
// was deleted and the tree left over after its deletion
func (m *StringMap) deleteLeftmost() (*StringMap, *StringMap) {
	if m.isLeaf() {
		return m, nilMap
	}

	for i, t := range m.children {
		if t != nilMap {
			deleted, child := t.deleteLeftmost()
			newMap := m.clone()
			newMap.children[i] = child
			recalculateCount(newMap)
			return deleted, newMap
		}
	}
	panic("Tree isn't a leaf but also had no children. How does that happen?")
}

// isLeaf returns true if this is a leaf node
func (m *StringMap) isLeaf() bool {
	return m.Size() == 1
}

// returns the number of child subtrees we have
func (m *StringMap) subtreeCount() int {
	count := 0
	for _, t := range m.children {
		if t != nilMap {
			count++
		}
	}
	return count
}

func (m *StringMap) Lookup(key string) (string, bool) {
	hash := hashKey(key)
	return lookupLowLevel(m, hash, hash)
}

func lookupLowLevel(self *StringMap, partialHash, hash uint64) (string, bool) {
	if self.IsNil() { // an empty tree is easy
		return nil, false
	}

	if hash != self.hash {
		i := partialHash % childCount
		return lookupLowLevel(self.children[i], partialHash>>shiftSize, hash)
	}

	// we found it
	return self.value, true
}

func (m *StringMap) Size() int {
	return m.count
}

func (m *StringMap) ForEach(f func(key string, val string)) {
	if m.IsNil() {
		return
	}

	// ourself
	f(m.key, m.value)

	// children
	for _, t := range m.children {
		if t != nilMap {
			t.ForEach(f)
		}
	}
}

func (m *StringMap) Keys() []string {
	keys := make([]string, m.Size())
	i := 0
	m.ForEach(func(k string, v string) {
		keys[i] = k
		i++
	})
	return keys
}

// make it easier to display maps for debugging
func (m *StringMap) String() string {
	keys := m.Keys()
	buf := bytes.NewBufferString("{")
	for _, key := range keys {
		val, _ := m.Lookup(key)
		fmt.Fprintf(buf, "%s: %s, ", key, val)
	}
	fmt.Fprintf(buf, "}\n")
	return buf.String()
}
