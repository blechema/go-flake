package flake

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_Demo(t *testing.T) {
	fmt.Printf("Sequence: %d bit, Clock: %.2f ms, Epoch: %.2f years (%s)\n",
		intervalBits, float64((time.Duration(1)<<ignoredTimeBits).Microseconds())/float64(1000),
		(time.Duration(1)<<(ignoredTimeBits+intervalBits)).Hours()/24/365,
		time.Unix(0, 0).Add(time.Duration(1)<<(ignoredTimeBits+intervalBits)))

	fmt.Println("--- Shuffled IDs ---")
	for i := int64(0); i < 4; i++ {
		id := Next()
		fmt.Println(id.Base64(), id.Base32(), id.Hex(), id)
		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("--- NextRaw IDs ---")
	for i := int64(0); i < 4; i++ {
		id := NextRaw()
		fmt.Println(id.Base64(), id.Base32(), id.Hex(), id)
	}
}

func TestWithMachineId(t *testing.T) {
	f := make([]Flaker, 256, 256)
	for i := 0; i < 256; i++ {
		f[i] = WithMachineId(byte(i))
	}
	m := make(map[Flake]int)
	for i := 0; i < 5000; i++ {
		for i := 0; i < 256; i++ {
			id := f[i].Next()
			if prev, ok := m[id]; ok {
				t.Errorf("doubble @ %d and %d with %d", i, prev, id)
				return
			}
			m[id] = i
		}
	}
}

func TestWithEpochStart(t *testing.T) {
	f1 := WithEpochStart(time.Unix(0, 0))
	f2 := WithEpochStart(time.Unix(1000, 0))
	m := make(map[Flake]int)
	generate(t, f1, m, 10000)
	generate(t, f2, m, 10000)
}

// This test generates 1,000,000 IDs and check for uniqueness
func TestSequencing(t *testing.T) {
	generate(t, Default, make(map[Flake]int), 5000000)
}

func TestEncode(t *testing.T) {
	in := Next()
	if out := in.Int64(); out != int64(in) {
		t.Errorf("Decoding of int64 value failed for input %d with output %d", in, out)
	}
	if out := in.Uint64(); out != uint64(in) {
		t.Errorf("Decoding of int64 value failed for input %d with output %d", in, out)
	}
	if out, err := Decode(in.Hex()); err != nil || out != in {
		t.Errorf("Decoding of hex value failed for input %d with output %d: %v", in, out, err)
	}
	if out, err := Decode(in.Base32()); err != nil || out != in {
		t.Errorf("Decoding of base32 value failed for input %d with output %d: %v", in, out, err)
	}
	if out, err := Decode(in.Base64()); err != nil || out != in {
		t.Errorf("Decoding of base64 value failed for input %d with output %d: %v", in, out, err)
	}
	if out, err := FromBytes(in.Bytes()); err != nil || out != in {
		t.Errorf("Decoding of bytes value failed for input %d with output %d: %v", in, out, err)
	}
}

func TestDecode(t *testing.T) {
	if _, err := Decode(""); err == nil { // unknown format
		t.Errorf("Decoding of string '' failed. No error!")
	}
	if _, err := Decode("1234567890"); err == nil { // unknown format
		t.Errorf("Decoding of string '1234567890' failed. No error!")
	}
	if _, err := Decode("§2345678901"); err == nil { // base64
		t.Errorf("Decoding of string '12345678901' failed. No error!")
	}
	if _, err := Decode("§234567890123"); err == nil { // base32
		t.Errorf("Decoding of string '1234567890123' failed. No error!")
	}
	if _, err := Decode("§234567890123456"); err == nil { // hex
		t.Errorf("Decoding of string '1234567890123' failed. No error!")
	}
	if _, err := FromBytes([]byte{0, 0, 0, 0}); err == nil { // hex
		t.Errorf("Decoding of bytes failed. No error!")
	}
}

func TestMultithreading(t *testing.T) {
	w := sync.WaitGroup{}
	ms := make([]map[Flake]int, 8, 8)
	for i := 0; i < 8; i++ {
		ms[i] = make(map[Flake]int)
		m := ms[i]
		w.Add(1)
		go func() {
			defer w.Done()
			generate(t, Default, m, 10000)
		}()
	}
	w.Wait()
	m := make(map[Flake]int)
	for i := 0; i < 8; i++ {
		for k, v := range ms[i] {
			if prev, ok := m[k]; ok {
				t.Errorf("doubble @ %d and %d with %d", v, prev, k)
				return
			}
			m[k] = v
		}
	}
}

func generate(t *testing.T, f Flaker, m map[Flake]int, count int) {
	for i := 0; i < count; i++ {
		id := f.Next()
		if prev, ok := m[id]; ok {
			t.Errorf("doubble @ %d and %d with %d", i, prev, id)
			return
		}
		m[id] = i
	}
}
