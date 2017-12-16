package main

import (
	"strconv"

	"github.com/tddhit/bindex"
	"github.com/tddhit/bindex/util"
)

func main() {
	b, err := bindex.New("../bindex.idx", false)
	if err != nil {
		panic(err)
	}
	for i := 1; i <= 10000000; i++ {
		util.LogInfo(string(b.Get([]byte("hello" + strconv.Itoa(i)))))
	}
	//for i := 1; i <= 10000000; i++ {
	//	b.Put([]byte("hello"+strconv.Itoa(i)), []byte("world"+strconv.Itoa(i)))
	//	util.LogInfo(i)
	//}
	//for i := 1; i <= 10000000; i++ {
	//	b.Delete([]byte("hello" + strconv.Itoa(i)))
	//	util.LogInfo(i)
	//}
}
