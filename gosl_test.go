package main

import "os"
import "testing"
import "github.com/dgraph-io/badger/y"

var Size int64 = 256 * 1024 * 1024

func TestMmap(t *testing.T) {
	t.Log("Trying mmap")
	flags := os.O_RDWR | os.O_CREATE | os.O_EXCL
	f, err := os.OpenFile("test.md", flags, 0666)
	defer f.Close()
	y.Check(err)
//	size := int64(256 * 1024 * 1024)
	t.Logf("Size is : %v", Size)
	_, err = y.Mmap(f,false,Size)
	if err != nil {
		t.Errorf("mmap failed with error: %v", err)
	}
}
