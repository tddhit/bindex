package bindex

import (
	"strconv"
	"testing"

	"github.com/tddhit/bindex/util"
)

func TestBIndex(t *testing.T) {
	b, err := New("bindex.idx", false)
	if err != nil {
		panic(err)
	}
	//for i := 1000; i <= 1200; i++ {
	//	b.Put([]byte("hello"+strconv.Itoa(i)), []byte("world"+strconv.Itoa(i)))
	//}
	//for i := 237; i <= 337; i++ {
	//	b.Delete([]byte("hello" + strconv.Itoa(i)))
	//}
	//for i := 1; i <= 111600; i++ {
	//	util.LogInfo(string(b.Get([]byte("hello" + strconv.Itoa(i)))))
	//}
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
	//util.LogDebug(string(b.Get([]byte("hello" + strconv.Itoa(20)))))
	util.LogDebug("")
}
