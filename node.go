package bindex

import (
	"bytes"
	"fmt"
	"sort"
	"unsafe"

	"github.com/tddhit/tools/log"
)

type node struct {
	bindex   *BIndex
	isLeaf   bool
	pgid     pgid
	parent   *node
	key      []byte
	children nodes
	inodes   inodes
}

func (n node) dump() {
	var typ = "branch"
	if n.isLeaf {
		typ = "leaf"
	}
	log.Debugf("[NODE %d {type=%s count=%d size=%d}]", n.pgid, typ, len(n.inodes), n.size())
	for _, item := range n.inodes {
		if n.isLeaf {
			log.Debugf("+L %s -> %s", item.key, item.value)
		} else {
			log.Debugf("+B %s -> %d", item.key, item.pgid)
		}
	}
}

func (n *node) minKeys() int {
	if n.isLeaf {
		return 1
	}
	return 2
}

func (n *node) size() int {
	sz, elsz := PageHeaderSize, n.pageElementSize()
	for i := 0; i < len(n.inodes); i++ {
		item := &n.inodes[i]
		sz += elsz + len(item.key) + len(item.value)
	}
	return sz
}

func (n *node) sizeLessThan(v int) bool {
	sz, elsz := PageHeaderSize, n.pageElementSize()
	for i := 0; i < len(n.inodes); i++ {
		item := &n.inodes[i]
		sz += elsz + len(item.key) + len(item.value)
		if sz >= v {
			return false
		}
	}
	return true
}

func (n *node) pageElementSize() int {
	if n.isLeaf {
		return LeafPageElementSize
	}
	return BranchPageElementSize
}

func (n *node) childAt(index int) *node {
	if n.isLeaf {
		panic(fmt.Sprintf("invalid childAt(%d) on a leaf node", index))
	}
	return n.bindex.node(n.inodes[index].pgid, n)
}

func (n *node) childIndex(child *node) int {
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i].key, child.key) != -1 })
	return index
}

func (n *node) nextSibling() *node {
	if n.parent == nil {
		return nil
	}
	index := n.parent.childIndex(n)
	if index >= len(n.parent.inodes)-1 {
		return nil
	}
	return n.parent.childAt(index + 1)
}

func (n *node) prevSibling() *node {
	if n.parent == nil {
		return nil
	}
	index := n.parent.childIndex(n)
	if index == 0 {
		return nil
	}
	log.Debug(n.parent)
	log.Debug(n)
	log.Debug(index)
	return n.parent.childAt(index - 1)
}

func (n *node) put(oldKey, newKey, value []byte, pgid pgid) {
	if len(oldKey) <= 0 {
		panic("put: zero-length old key")
	} else if len(newKey) <= 0 {
		panic("put: zero-length new key")
	}
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i].key, oldKey) != -1 })
	exact := (len(n.inodes) > 0 && index < len(n.inodes) && bytes.Equal(n.inodes[index].key, oldKey))
	if !exact {
		n.inodes = append(n.inodes, inode{})
		copy(n.inodes[index+1:], n.inodes[index:])
	}
	inode := &n.inodes[index]
	inode.key = newKey
	inode.value = value
	inode.pgid = pgid
	n.bindex.uncommited[n.pgid] = n
}

func (n *node) splitIndex() (index int) {
	sz := PageHeaderSize
	for i := 0; i < len(n.inodes); i++ {
		index = i
		inode := n.inodes[i]
		elsize := n.pageElementSize() + len(inode.key) + len(inode.value)
		if sz+elsize > n.size()/2 {
			break
		}
		sz += elsize
	}
	return
}

func (n *node) rebalanceAfterInsert() {
	log.Debug("-------rebalance start-----------", n)
	if n.sizeLessThan(n.bindex.pageSize) {
		log.Debug("AfterInsert return!", "size:", n.size)
		return
	}
	splitIndex := n.splitIndex()
	if n.parent == nil {
		n.parent = &node{
			bindex:   n.bindex,
			isLeaf:   false,
			pgid:     n.bindex.allocPage(),
			parent:   nil,
			children: nodes{n},
		}
		n.bindex.root = n.parent.pgid
		n.bindex.maxPgid = n.parent.pgid
		n.bindex.nodes[n.parent.pgid] = n.parent
		n.bindex.uncommited[n.parent.pgid] = n.parent
	}
	next := &node{
		bindex: n.bindex,
		isLeaf: n.isLeaf,
		pgid:   n.bindex.allocPage(),
		parent: n.parent,
	}
	next.inodes = make(inodes, len(n.inodes[splitIndex:]))
	copy(next.inodes, n.inodes[splitIndex:])
	if n.children != nil {
		next.children = make(nodes, len(n.children[splitIndex:]))
		copy(next.children, n.children[splitIndex:])
	}
	for i := 0; i < len(next.inodes); i++ {
		if n := n.bindex.nodes[next.inodes[i].pgid]; n != nil {
			n.parent = next
		}
	}
	n.inodes = n.inodes[:splitIndex]
	if n.children != nil {
		n.children = n.children[:splitIndex]
	}
	n.parent.children = append(n.parent.children, next)
	var key = n.key
	if key == nil {
		key = n.inodes[0].key
	}
	n.parent.put(key, n.inodes[0].key, nil, n.pgid)
	n.key = n.inodes[0].key
	key = next.key
	if key == nil {
		key = next.inodes[0].key
	}
	n.parent.put(key, next.inodes[0].key, nil, next.pgid)
	next.key = next.inodes[0].key
	n.bindex.nodes[next.pgid] = next
	n.bindex.uncommited[n.pgid] = n
	n.bindex.uncommited[next.pgid] = next
	n.bindex.uncommited[n.parent.pgid] = n.parent
	log.Debug("-----------------------rebalance end--------------------")
	n.parent.rebalanceAfterInsert()
}

func (n *node) del(key []byte) {
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i].key, key) != -1 })
	if index >= len(n.inodes) || !bytes.Equal(n.inodes[index].key, key) {
		return
	}
	log.Debug(n.pgid, " del ", n.inodes[index].pgid, string(key))
	n.inodes = append(n.inodes[:index], n.inodes[index+1:]...)
	n.bindex.uncommited[n.pgid] = n
	n.dump()
}

func (n *node) rebalanceAfterDelete() {
	log.Debug("-----------------------rebalance start------", n)
	var threshold = n.bindex.pageSize / 2
	if !n.sizeLessThan(threshold) && len(n.inodes) > n.minKeys() {
		log.Debug("AfterDelete return!", "pgid:", n.pgid, "size:", n.size(), "inodes:", len(n.inodes))
		return
	}
	if n.parent == nil {
		log.Debug("n.parent == nil return")
		if !n.isLeaf && len(n.inodes) == 1 {
			log.Debug("root.inodes=1")
			child := n.bindex.node(n.inodes[0].pgid, n)
			n.isLeaf = child.isLeaf
			n.inodes = child.inodes[:]
			n.children = child.children
			n.key = child.key
			for _, inode := range n.inodes {
				if child, ok := n.bindex.nodes[inode.pgid]; ok {
					child.parent = n
				}
			}
			delete(n.bindex.nodes, child.pgid)
			delete(n.bindex.uncommited, child.pgid)
			n.bindex.uncommited[n.pgid] = n
		}
		log.Debug("-----------------------rebalance end--------------------")
		return
	}
	if len(n.inodes) == 0 {
		log.Debug("pgid:", n.pgid, " inodes=0 ")
		n.parent.del(n.key)
		n.parent.removeChild(n)
		delete(n.bindex.nodes, n.pgid)
		delete(n.bindex.uncommited, n.pgid)
		log.Debug("-----------------------rebalance end--------------------")
		n.parent.rebalanceAfterDelete()
		return
	}
	var target *node
	var useNextSibling = (n.parent.childIndex(n) == 0)
	if useNextSibling {
		target = n.nextSibling()
	} else {
		target = n.prevSibling()
	}
	if target == nil {
		log.Debug("target=nil:", target == nil)
		log.Debug("-----------------------rebalance end--------------------")
		n.parent.rebalanceAfterDelete()
		return
	}
	if target.size()+n.size() > n.bindex.pageSize {
		log.Debug(" size > pageSize:", target.size(), n.size())
		log.Debug("-----------------------rebalance end--------------------")
		n.parent.rebalanceAfterDelete()
		return
	}
	log.Debug("merge:", n.pgid, target.pgid)
	if useNextSibling {
		for _, inode := range target.inodes {
			if child, ok := n.bindex.nodes[inode.pgid]; ok {
				child.parent.removeChild(child)
				child.parent = n
				child.parent.children = append(child.parent.children, child)
			}
		}
		n.inodes = append(n.inodes, target.inodes...)
		n.parent.del(target.key)
		n.parent.removeChild(target)
		delete(n.bindex.nodes, target.pgid)
		delete(n.bindex.uncommited, target.pgid)
		n.bindex.uncommited[n.pgid] = n
	} else {
		for _, inode := range n.inodes {
			if child, ok := n.bindex.nodes[inode.pgid]; ok {
				child.parent.removeChild(child)
				child.parent = target
				child.parent.children = append(child.parent.children, child)
			}
		}
		target.inodes = append(target.inodes, n.inodes...)
		n.parent.del(n.key)
		n.parent.removeChild(n)
		delete(n.bindex.nodes, n.pgid)
		delete(n.bindex.uncommited, n.pgid)
		n.bindex.uncommited[target.pgid] = target
	}
	n.bindex.uncommited[n.parent.pgid] = n.parent
	log.Debug("-----------------------rebalance end--------------------")
	n.parent.rebalanceAfterDelete()
}

func (n *node) read(p *page) {
	n.pgid = p.id
	n.isLeaf = ((p.flags & LeafPageFlag) != 0)
	n.inodes = make(inodes, int(p.count))
	for i := 0; i < int(p.count); i++ {
		inode := &n.inodes[i]
		if n.isLeaf {
			elem := p.leafPageElement(uint16(i))
			//inode.key = elem.key()
			//inode.value = elem.value()
			key := make([]byte, len(elem.key()))
			copy(key, elem.key())
			inode.key = key
			value := make([]byte, len(elem.value()))
			copy(value, elem.value())
			inode.value = value
		} else {
			elem := p.branchPageElement(uint16(i))
			key := make([]byte, len(elem.key()))
			copy(key, elem.key())
			inode.key = key
			inode.pgid = elem.pgid
		}
	}
	if len(n.inodes) > 0 {
		n.key = n.inodes[0].key
	} else {
		n.key = nil
	}
}

func (n *node) write() error {
	//buf := n.bindex.pagePool.Get().([]byte)
	buf := make([]byte, n.bindex.pageSize)
	p := n.bindex.pageInBuffer(buf, pgid(0))
	p.id = n.pgid
	if n.isLeaf {
		p.flags |= LeafPageFlag
	} else {
		p.flags |= BranchPageFlag
	}
	p.count = uint16(len(n.inodes))
	if p.count == 0 {
		//n.bindex.pagePool.Put(buf)
	}
	b := (*[MaxMapSize]byte)(unsafe.Pointer(&p.ptr))[n.pageElementSize()*len(n.inodes):]
	for i, item := range n.inodes {
		if n.isLeaf {
			elem := p.leafPageElement(uint16(i))
			elem.pos = uint32(uintptr(unsafe.Pointer(&b[0])) - uintptr(unsafe.Pointer(elem)))
			elem.ksize = uint32(len(item.key))
			elem.vsize = uint32(len(item.value))
		} else {
			elem := p.branchPageElement(uint16(i))
			elem.pos = uint32(uintptr(unsafe.Pointer(&b[0])) - uintptr(unsafe.Pointer(elem)))
			elem.ksize = uint32(len(item.key))
			elem.pgid = item.pgid
		}
		klen, vlen := len(item.key), len(item.value)
		if len(b) < klen+vlen {
			b = (*[MaxMapSize]byte)(unsafe.Pointer(&b[0]))[:]
		}
		copy(b[0:], item.key)
		b = b[klen:]
		copy(b[0:], item.value)
		b = b[vlen:]
	}
	log.Debug("n.size:", n.size())
	p.dump()
	//ptr := (*[MaxMapSize]byte)(unsafe.Pointer(p))
	//buff := ptr[:n.bindex.pageSize]
	offset := int64(n.pgid) * int64(n.bindex.pageSize)
	if _, err := n.bindex.file.WriteAt(buf, offset); err != nil {
		return err
	}
	return nil
}

func (n *node) removeChild(target *node) {
	for i, child := range n.children {
		if child == target {
			n.children = append(n.children[:i], n.children[i+1:]...)
			return
		}
	}
}

func (n *node) dereference() {
	if n.key != nil {
		key := make([]byte, len(n.key))
		copy(key, n.key)
		n.key = key
	}
	for i := range n.inodes {
		inode := &n.inodes[i]
		key := make([]byte, len(inode.key))
		copy(key, inode.key)
		inode.key = key
		value := make([]byte, len(inode.value))
		copy(value, inode.value)
		inode.value = value
	}
	for _, child := range n.children {
		child.dereference()
	}
}

type nodes []*node

func (s nodes) Len() int           { return len(s) }
func (s nodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s nodes) Less(i, j int) bool { return bytes.Compare(s[i].inodes[0].key, s[j].inodes[0].key) == -1 }

type inode struct {
	pgid  pgid
	key   []byte
	value []byte
}

type inodes []inode
