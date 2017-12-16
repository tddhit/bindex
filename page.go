package bindex

import (
	"unsafe"

	"github.com/tddhit/bindex/util"
)

const (
	PageHeaderSize        = int(unsafe.Offsetof(((*page)(nil)).ptr))
	BranchPageElementSize = int(unsafe.Sizeof(branchPageElement{}))
	LeafPageElementSize   = int(unsafe.Sizeof(leafPageElement{}))
)

const (
	BranchPageFlag = 0x01
	LeafPageFlag   = 0x02
	MetaPageFlag   = 0x04
)

type pgid uint64

type page struct {
	id    pgid
	flags uint16
	count uint16
	ptr   uintptr
}

func (p *page) dump() {
	util.LogDebug("-------------------------")
	util.LogDebug("pgid:", p.id)
	util.LogDebug("flag:", p.flags)
	util.LogDebug("count:", p.count)
	for i := 0; i < int(p.count); i++ {
		if p.flags == LeafPageFlag {
			elem := p.leafPageElement(uint16(i))
			util.LogDebug("key:", string(elem.key()))
			util.LogDebug("value:", string(elem.value()))
		} else {
			elem := p.branchPageElement(uint16(i))
			util.LogDebug("key:", string(elem.key()))
			util.LogDebug("pgid:", elem.pgid)
		}
	}
}

func (p *page) meta() *meta {
	return (*meta)(unsafe.Pointer(&p.ptr))
}

func (p *page) leafPageElement(index uint16) *leafPageElement {
	n := &((*[0x7FFFFFF]leafPageElement)(unsafe.Pointer(&p.ptr)))[index]
	return n
}

func (p *page) leafPageElements() []leafPageElement {
	if p.count == 0 {
		return nil
	}
	return ((*[0x7FFFFFF]leafPageElement)(unsafe.Pointer(&p.ptr)))[:]
}

func (p *page) branchPageElement(index uint16) *branchPageElement {
	return &((*[0x7FFFFFF]branchPageElement)(unsafe.Pointer(&p.ptr)))[index]
}

func (p *page) branchPageElements() []branchPageElement {
	if p.count == 0 {
		return nil
	}
	return ((*[0x7FFFFFF]branchPageElement)(unsafe.Pointer(&p.ptr)))[:]
}

type pages []*page

func (s pages) Len() int           { return len(s) }
func (s pages) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s pages) Less(i, j int) bool { return s[i].id < s[j].id }

type branchPageElement struct {
	pos   uint32
	ksize uint32
	pgid  pgid
}

func (n *branchPageElement) key() []byte {
	buf := (*[MaxMapSize]byte)(unsafe.Pointer(n))
	return (*[MaxMapSize]byte)(unsafe.Pointer(&buf[n.pos]))[:n.ksize]
}

type leafPageElement struct {
	pos   uint32
	ksize uint32
	vsize uint32
}

func (n *leafPageElement) key() []byte {
	buf := (*[MaxMapSize]byte)(unsafe.Pointer(n))
	return (*[MaxMapSize]byte)(unsafe.Pointer(&buf[n.pos]))[:n.ksize:n.ksize]
}

func (n *leafPageElement) value() []byte {
	buf := (*[MaxMapSize]byte)(unsafe.Pointer(n))
	return (*[MaxMapSize]byte)(unsafe.Pointer(&buf[n.pos+n.ksize]))[:n.vsize:n.vsize]
}
