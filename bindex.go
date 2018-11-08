package bindex

import (
	"bytes"
	"container/list"
	"errors"
	"hash/fnv"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"
)

var (
	ErrKeyRequired     = errors.New("key required")
	ErrKeyTooLarge     = errors.New("key too large")
	ErrValueTooLarge   = errors.New("value too large")
	ErrVersionMismatch = errors.New("version mismatch")
	ErrChecksum        = errors.New("checksum error")
	ErrInvalid         = errors.New("invalid bindex")
	ErrFileTooLarge    = errors.New("file size greater than MaxMapSize")
)

const (
	VERSION      = 2
	MAGIC        = 0x12345678
	maxMapSize   = 1 << 37 //128G
	MaxKeySize   = 128
	MaxValueSize = 128
)

type BIndex struct {
	file       *mmap.MmapFile
	nodes      map[pgid]*node
	uncommited map[pgid]*node
	pagePool   sync.Pool
	pageSize   int
	root       pgid
	maxPgid    pgid
}

type meta struct {
	magic    uint32
	version  uint32
	pageSize uint32
	root     pgid
	maxPgid  pgid
	checksum uint64
}

func New(path string, mode, advise int) (*BIndex, error) {
	file, err := mmap.New(path, maxMapSize, mode, advise)
	if err != nil {
		return nil, err
	}
	b := &BIndex{
		file:       file,
		nodes:      make(map[pgid]*node),
		uncommited: make(map[pgid]*node),
		pageSize:   os.Getpagesize(),
	}
	b.pagePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, b.pageSize)
		},
	}
	if b.file.Size() == 0 {
		if err := b.init(); err != nil {
			return nil, err
		}
	} else if b.file.Size() > maxMapSize {
		return nil, ErrFileTooLarge
	}
	meta := b.page(0).meta()
	b.root = meta.root
	b.maxPgid = meta.maxPgid
	return b, nil
}

func (b *BIndex) Close() error {
	return b.file.Close()
}

func (b *BIndex) allocPage() pgid {
	return pgid(atomic.AddUint64((*uint64)(unsafe.Pointer(&b.maxPgid)), 1))
}

func (b *BIndex) init() error {
	buf := make([]byte, b.pageSize*2)
	p := b.pageInBuffer(buf[:], pgid(0))
	p.id = pgid(0)
	p.flags = MetaPageFlag
	m := p.meta()
	m.magic = MAGIC
	m.version = VERSION
	m.pageSize = uint32(b.pageSize)
	m.root = 1
	m.maxPgid = 1
	m.checksum = m.sum64()

	p = b.pageInBuffer(buf[:], pgid(1))
	p.id = pgid(1)
	p.flags = LeafPageFlag
	p.count = 0

	if err := b.file.WriteAt(buf, 0); err != nil {
		return err
	}
	if err := b.file.Sync(); err != nil {
		return err
	}
	return nil
}

func (b *BIndex) pageInBuffer(buf []byte, id pgid) *page {
	return (*page)(unsafe.Pointer(&buf[id*pgid(b.pageSize)]))
}

func (b *BIndex) dump() {
	l := list.New()
	root := b.node(b.root, nil)
	if root == nil {
		return
	}
	l.PushBack(root)
	for {
		if l.Len() > 0 {
			e := l.Front()
			n := e.Value.(*node)
			n.dump()
			l.Remove(e)
			for i := 0; i < len(n.children); i++ {
				l.PushBack(n.children[i])
			}
		} else {
			break
		}
	}
}

func (b *BIndex) Put(key []byte, value []byte) error {
	log.Debug("Put:", string(key))
	if len(key) == 0 {
		return ErrKeyRequired
	} else if len(key) > MaxKeySize {
		return ErrKeyTooLarge
	} else if int64(len(value)) > MaxValueSize {
		return ErrValueTooLarge
	}
	c := b.NewCursor()
	c.seek(key)
	var clone = make([]byte, len(key))
	copy(clone, key)
	n := c.node()
	n.put(clone, clone, value, 0)
	n.rebalanceAfterInsert()
	minNode := b.minNode()
	log.Debug("minNode:", minNode)
	if len(minNode.inodes) > 0 && bytes.Compare(minNode.inodes[0].key, key) > 0 {
		b.adjustMinKey(minNode, minNode.inodes[0].key)
	}
	b.commit()
	return nil
}

func (b *BIndex) Get(key []byte) []byte {
	c := b.NewCursor()
	b.dump()
	k, v := c.seek(key)
	if !bytes.Equal(key, k) {
		return nil
	}
	return v
}

func (b *BIndex) Delete(key []byte) error {
	log.Debug("Delete:", string(key))
	c := b.NewCursor()
	c.seek(key)
	n := c.node()
	n.del(key)
	n.rebalanceAfterDelete()
	minNode := b.minNode()
	log.Debug("minNode:", minNode)
	if len(minNode.inodes) > 0 && bytes.Compare(minNode.inodes[0].key, key) > 0 {
		b.adjustMinKey(minNode, minNode.inodes[0].key)
	}
	for _, e := range c.stack {
		var pgid pgid
		if e.node != nil {
			pgid = e.node.pgid
		} else if e.page != nil {
			pgid = e.page.id
		}
		if n, ok := b.nodes[pgid]; ok {
			n.dump()
		} else {
			log.Debug(pgid, "is not exist b.nodes!")
		}
	}
	b.commit()
	return nil
}

func (b *BIndex) NewCursor() *Cursor {
	c := &Cursor{
		bindex: b,
		stack:  make([]elemRef, 0),
	}
	return c
}

func (b *BIndex) minNode() *node {
	var minNode *node
	n := b.node(b.root, nil)
	for {
		if n.isLeaf {
			minNode = n
			log.Debug("minNode:", minNode.pgid)
			break
		} else {
			n = n.childAt(0)
		}
	}
	return minNode
}

func (b *BIndex) adjustMinKey(minNode *node, key []byte) {
	n := minNode
	for {
		if n.parent == nil {
			break
		} else {
			n = n.parent
			copy(n.inodes[0].key, key)
			n.key = n.inodes[0].key
			log.Debug("replace min:", n.pgid, n.inodes)
			b.uncommited[n.pgid] = n
		}
	}
}

func (b *BIndex) node(pgid pgid, parent *node) *node {
	if n := b.nodes[pgid]; n != nil {
		return n
	}
	n := &node{
		bindex: b,
		parent: parent,
	}
	if parent == nil {
		b.root = pgid
	} else {
		parent.children = append(parent.children, n)
	}
	var p = b.page(pgid)
	n.read(p)
	b.nodes[pgid] = n
	return n
}

func (b *BIndex) page(id pgid) *page {
	pos := id * pgid(b.pageSize)
	return (*page)(b.file.Buf(int64(pos)))
}

func (b *BIndex) pageNode(id pgid) (*page, *node) {
	if b.nodes != nil {
		if n := b.nodes[id]; n != nil {
			return nil, n
		}
	}
	return b.page(id), nil
}

func (b *BIndex) commit() error {
	log.Debug("commit start:", b.uncommited)
	//buf := make([]byte, b.pageSize)
	buf := b.pagePool.Get().([]byte)
	p := b.pageInBuffer(buf, pgid(0))
	p.id = pgid(0)
	p.flags = MetaPageFlag
	m := p.meta()
	m.magic = MAGIC
	m.version = VERSION
	m.pageSize = uint32(b.pageSize)
	m.root = b.root
	m.maxPgid = b.maxPgid
	m.checksum = m.sum64()
	if err := b.file.WriteAt(buf, 0); err != nil {
		b.pagePool.Put(buf)
		return err
	}
	b.pagePool.Put(buf)
	for _, node := range b.uncommited {
		log.Debug(node)
		//node.dereference()
		node.write()
		log.Debug(node)
	}
	if err := b.file.Sync(); err != nil {
		return err
	}
	b.uncommited = make(map[pgid]*node)
	log.Debug("commit end")
	return nil
}

func (m *meta) sum64() uint64 {
	var h = fnv.New64a()
	_, _ = h.Write((*[unsafe.Offsetof(meta{}.checksum)]byte)(unsafe.Pointer(m))[:])
	return h.Sum64()
}

func (m *meta) validate() error {
	if m.magic != MAGIC {
		return ErrInvalid
	} else if m.version != VERSION {
		return ErrVersionMismatch
	} else if m.checksum != 0 && m.checksum != m.sum64() {
		return ErrChecksum
	}
	return nil
}

func (b *BIndex) Traverse() {
	l := list.New()
	root := b.node(b.root, nil)
	if root == nil {
		return
	}
	l.PushBack(root)
	for {
		if l.Len() > 0 {
			e := l.Front()
			n := e.Value.(*node)
			if n.isLeaf {
				n.dump()
			} else {
				for i := 0; i < len(n.inodes); i++ {
					node := b.node(n.inodes[i].pgid, n)
					l.PushBack(node)
				}
			}
			l.Remove(e)
		} else {
			break
		}
	}
}
