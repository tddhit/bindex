package main

import (
	"strconv"

	"github.com/tddhit/tools/log"

	"github.com/tddhit/bindex"
)

func main() {
	b, err := bindex.New("../bindex.idx", false)
	if err != nil {
		panic(err)
	}
	for i := 1; i <= 10000000; i++ {
		log.Info(string(b.Get([]byte("hello" + strconv.Itoa(i)))))
	}
	//for i := 1; i <= 10000000; i++ {
	//	b.Put([]byte("hello"+strconv.Itoa(i)), []byte("world"+strconv.Itoa(i)))
	//	log.Info(i)
	//}
	//for i := 1; i <= 10000000; i++ {
	//	b.Delete([]byte("hello" + strconv.Itoa(i)))
	//	log.Info(i)
	//}
}
