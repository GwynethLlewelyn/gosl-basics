// Minimalist test to confirm that basic functionality works.
// This can be safely ignored.
package main

import (
	"fmt"
	"log"

	badger "github.com/dgraph-io/badger/v3"
)

func main() {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	//db, err := badger.Open(badger.DefaultOptions("slkvdb/gosl-database.db"))
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(txn *badger.Txn) error {
		err = txn.Set([]byte("answer"), []byte("42"))
		return err
	})
	handle(err)

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("answer"))
		handle(err)

		var valCopy []byte
		err = item.Value(func(val []byte) error {
			// This func with val would only be called if item.Value encounters no error.

			// Accessing val here is valid.
			fmt.Printf("(Inside item.Value()) The answer is: %s\n", val)

			// Copying or parsing val is valid.
			valCopy = append([]byte{}, val...)

			return nil
		})
		handle(err)

		// You must copy it to use it outside item.Value(...).
		fmt.Printf("(inside db.View()) The answer is: %s\n", valCopy)

		// Alternatively, you could also use item.ValueCopy().
		valCopy, err = item.ValueCopy(nil)
		handle(err)
		fmt.Printf("(inside db.View() using ValueCopy) The answer is: %s\n", valCopy)

		return nil
	})
	handle(err)
}

// Handles errors for this very simple test.
func handle(err error) {
	fmt.Printf("error was: %q\n", err)
}