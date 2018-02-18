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

func BenchmarkCopy(b *testing.B) {
	var data = make([]byte, 1024)
	var data2 = make([]byte, 1024)
	for n := 0; n < b.N; n++ {
		for i, v := range data {
			data2[i] = v
		}
	}
	//log.Printf("%v", hex.Dump(data2))
}

func BenchmarkCopy2(b *testing.B) {
	var data = make([]byte, 1024)
	var data2 = make([]byte, 1024)
	for n := 0; n < b.N; n++ {
		copy(data2, data)
	}
	//log.Printf("%v", hex.Dump(data2))
}

func BenchmarkCopy3(b *testing.B) {
	var data = make([]byte, 1024)
	var data2 = make([]byte, 1028)
	for n := 0; n < b.N; n++ {
		copy(data2[4:(len(data)+4)], data)
	}
	//log.Printf("%v", hex.Dump(data2))
}

func BenchmarkCopy4(b *testing.B) { // faster, slightly
	var data = make([]byte, 1024)
	for n := 0; n < b.N; n++ {
		var data2 = make([]byte, len(data)+4)
		copy(data2[4:], data)
	}
	//log.Printf("%v", hex.Dump(data2))
}

func BenchmarkCopy5(b *testing.B) { // slower
	var data = make([]byte, 1024)
	for n := 0; n < b.N; n++ {
		data2 := append([]byte{0, 0, 0, 0}, data...)
		_ = data2
	}
	//log.Printf("%v", hex.Dump(data2))
}

func BenchmarkEscapeTest(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var data = make([]byte, 1024)
		length := (len(data)+2)*2 + 2
		currentPos := 0
		outputBytes := make([]byte, length)
		fcs := pppInitFCS16
		for _, v := range data {
			// black magic
			fcs = fcs>>8 ^ int(fcstab[(fcs^int(v))&0xff])
			// escape byte
			if v < 0x20 || v == flagSequence || v == controlEscape {
				outputBytes[currentPos] = controlEscape
				currentPos++
				outputBytes[currentPos] = v ^ 0x20
				currentPos++
			} else {
				outputBytes[currentPos] = v
				currentPos++
			}
		}
	}
}

func BenchmarkEscapeTest2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var data = make([]byte, 1024)
		length := (len(data)+2)*2 + 2
		currentPos := 0
		outputBytes := make([]byte, length)
		var fcs uint16 = pppInitFCS16
		for _, v := range data {
			// black magic
			fcs = fcs>>8 ^ fcstab[(fcs^uint16(v))&0xff]
			// escape byte
			if v < 0x20 || v == flagSequence || v == controlEscape {
				outputBytes[currentPos] = controlEscape
				currentPos++
				outputBytes[currentPos] = v ^ 0x20
				currentPos++
			} else {
				outputBytes[currentPos] = v
				currentPos++
			}
		}
	}
}
