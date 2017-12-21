package bindex

import (
	"bytes"
	"sort"

	"github.com/tddhit/tools/log"
)

type Cursor struct {
	bindex *BIndex
	stack  []elemRef
}

func (c *Cursor) seek(seek []byte) (key []byte, value []byte) {
	c.search(seek, c.bindex.root)
	for i := 0; i < len(c.stack); i++ {
		e := c.stack[i]
		if e.page != nil {
			log.Debug("seek:", e.page.id)
		} else if e.node != nil {
			log.Debug("seek:", e.node.pgid)
		}
	}
	ref := &c.stack[len(c.stack)-1]
	if ref.count() == 0 || ref.index >= ref.count() {
		return nil, nil
	}
	if ref.node != nil {
		inode := &ref.node.inodes[ref.index]
		return inode.key, inode.value
	}
	elem := ref.page.leafPageElement(uint16(ref.index))
	return elem.key(), elem.value()
}

func (c *Cursor) search(key []byte, pgid pgid) {
	p, n := c.bindex.pageNode(pgid)
	e := elemRef{page: p, node: n}
	c.stack = append(c.stack, e)
	if e.isLeaf() {
		c.nsearch(key)
		return
	}
	if n != nil {
		c.searchNode(key, n)
		return
	}
	c.searchPage(key, p)
}

func (c *Cursor) searchNode(key []byte, n *node) {
	var exact bool
	index := sort.Search(len(n.inodes), func(i int) bool {
		ret := bytes.Compare(n.inodes[i].key, key)
		if ret == 0 {
			exact = true
		}
		return ret != -1
	})
	if !exact && index > 0 {
		index--
	}
	log.Debug("searchNode:", n, index, string(key))
	c.stack[len(c.stack)-1].index = index
	c.search(key, n.inodes[index].pgid)
}

func (c *Cursor) searchPage(key []byte, p *page) {
	inodes := p.branchPageElements()
	var exact bool
	index := sort.Search(int(p.count), func(i int) bool {
		ret := bytes.Compare(inodes[i].key(), key)
		if ret == 0 {
			exact = true
		}
		return ret != -1
	})
	if !exact && index > 0 {
		index--
	}
	c.stack[len(c.stack)-1].index = index
	c.search(key, inodes[index].pgid)
}

func (c *Cursor) nsearch(key []byte) {
	e := &c.stack[len(c.stack)-1]
	p, n := e.page, e.node
	if n != nil {
		index := sort.Search(len(n.inodes), func(i int) bool {
			return bytes.Compare(n.inodes[i].key, key) != -1
		})
		e.index = index
		return
	}
	inodes := p.leafPageElements()
	index := sort.Search(int(p.count), func(i int) bool {
		return bytes.Compare(inodes[i].key(), key) != -1
	})
	e.index = index
}

func (c *Cursor) node() *node {
	if ref := &c.stack[len(c.stack)-1]; ref.node != nil && ref.isLeaf() {
		return ref.node
	}
	var n = c.stack[0].node
	if n == nil {
		n = c.bindex.node(c.stack[0].page.id, nil)
	}
	for _, ref := range c.stack[:len(c.stack)-1] {
		n = n.childAt(int(ref.index))
	}
	return n
}

type elemRef struct {
	page  *page
	node  *node
	index int
}

func (r *elemRef) isLeaf() bool {
	if r.node != nil {
		return r.node.isLeaf
	}
	return (r.page.flags & LeafPageFlag) != 0
}

func (r *elemRef) count() int {
	if r.node != nil {
		return len(r.node.inodes)
	}
	return int(r.page.count)
}
