package main

import (
	"testing"
)

type logWriter struct{}

func (l logWriter) Write(data []byte) (int, error) {
	//log.Printf("%v", data)
	return len(data), nil
}

func BenchmarkEscape(b *testing.B) {
	var data = make([]byte, 1024)
	writer := logWriter{}
	unescaper := newUnescaper(writer)
	var data2 []byte
	for n := 0; n < b.N; n++ {
		data2 = pppEscape(data)
		unescaper.Write(data2)
	}
	//log.Printf("%v", hex.Dump(data2))
}
