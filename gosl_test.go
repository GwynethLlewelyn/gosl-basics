// gosl_test just does some simple mmap tests for Badger.
// You can safely ignore it for now.
package main

import "os"
import "testing"
import "github.com/dgraph-io/badger/y"

var Size int64 = 128 * 1024 * 1024

func TestMmap(t *testing.T) {
	t.Log("Trying mmap")
	var i int64
	for i = 1; i < 8; i++ {
		flags := os.O_RDWR | os.O_CREATE | os.O_EXCL
		f, err := os.OpenFile("test.md", flags, 0666)
		defer f.Close()
		y.Check(err)
	//	size := int64(256 * 1024 * 1024)
		t.Logf("Size is : %v", i*Size)
		_, err = y.Mmap(f,false,i*Size)
		if err != nil {
			t.Errorf("mmap failed with error: %v", err)
		}
		err = os.Remove("test.md")
		if err != nil {
			t.Errorf("could not remove test.md: %v", err)
		}
	}
}
