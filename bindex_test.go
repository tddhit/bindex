package bindex

import (
	"strconv"
	"testing"

	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"
)

func TestBIndex(t *testing.T) {
	log.Init("", log.INFO)
	b, err := New("bindex.idx", mmap.CREATE, mmap.RANDOM)
	if err != nil {
		panic(err)
	}
	for i := 10; i <= 12; i++ {
		b.Put([]byte("hello"+strconv.Itoa(i)), []byte("world"+strconv.Itoa(i)))
	}
	//for i := 237; i <= 337; i++ {
	//	b.Delete([]byte("hello" + strconv.Itoa(i)))
	//}
	//for i := 1000; i <= 1200; i++ {
	//	log.Info(string(b.Get([]byte("hello" + strconv.Itoa(i)))))
	//}
	c := b.NewCursor()
	k, v := c.First()
	log.Info(string(k), string(v))
	k, v = c.Last()
	log.Info(string(k), string(v))
	for k, v := c.First(); k != nil; k, v = c.Next() {
		log.Info(string(k), string(v))
	}
	//for i := 1100; i <= 1180; i++ {
	//	b.Put([]byte("hello"+strconv.Itoa(i)), []byte("world"+strconv.Itoa(i)))
	//}
	//for i := 1; i <= 11111500; i++ {
	//	b.Put([]byte("hello"+strconv.Itoa(i)), []byte("world"+strconv.Itoa(i)))
	//}
	//for i := 16; i <= 20; i++ {
	//	b.Delete([]byte("hello" + strconv.Itoa(i)))
	//}
	//b.Delete([]byte("hello" + strconv.Itoa(20)))
	//log.Debug(string(b.Get([]byte("hello" + strconv.Itoa(20)))))
	log.Debug("")
}
