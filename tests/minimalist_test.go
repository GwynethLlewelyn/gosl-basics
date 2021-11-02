// Minimalist test to confirm that basic functionality works.
package main

import (
	"os"
	"testing"

	badger "github.com/dgraph-io/badger/v3"
)

func TestMinimalistSLKVDB(t *testing.T) {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	// db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	t.Log("Testing basic Badger functionality with a simple database read")
	db, err := badger.Open(badger.DefaultOptions("slkvdb/gosl-database.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()	// First close database,
		// then remove all the directories and files
		// Using RemoveAll() function
		if err = os.RemoveAll("slkvdb"); err != nil {
			t.Fatal(err)
		} else {
			t.Log("cleaning up test database successful")
		}
	}()

	err = db.Update(func(txn *badger.Txn) error {
		err = txn.Set([]byte("answer"), []byte("42"))
		return err
	})
	handle(t, err)

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("answer"))
		handle(t, err)

		var valCopy []byte
		err = item.Value(func(val []byte) error {
			// This func with val would only be called if item.Value encounters no error.

			// Accessing val here is valid.
			t.Logf("(Inside item.Value()) The answer is: %q\n", val)

			// Copying or parsing val is valid.
			valCopy = append([]byte{}, val...)

			return nil
		})
		handle(t, err)

		// You must copy it to use it outside item.Value(...).
		t.Logf("(inside db.View()) The answer is: %q\n", valCopy)

		// Alternatively, you could also use item.ValueCopy().
		valCopy, err = item.ValueCopy(nil)
		handle(t, err)
		t.Logf("(inside db.View() using ValueCopy) The answer is: %s\n", valCopy)

		return nil
	})
	handle(t, err)
}

func TestRealDataseSLKVDB(t *testing.T) {	// Open the Badger database located in the /tmp/badger directory.
	t.Log("Testing Badger functionality reading actual database (first 20 entries only)")
	db, err := badger.Open(badger.DefaultOptions("../slkvdb/gosl-database.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		maxItems := 0	// to display only the first N entries (gwyneth 20211102)
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				t.Logf("key=%s, value=%s\n", k, v)
				return nil
			})
			if err != nil {
				return err
			}
			if maxItems > 20 {
				break
			}
			maxItems++
		}
		t.Logf("%d item(s) read.\n", maxItems)
		return nil
		})
	handle(t, err)
}


// Handles errors for this very simple test.
func handle(t *testing.T, err error) {
	if err != nil {
		t.Logf("error was: %q\n", err)
	}
}