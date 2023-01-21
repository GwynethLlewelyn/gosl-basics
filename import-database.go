// Tools to import a avatar key & name database in CSV format into a KV database.
package main

import (
	"compress/bzip2"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tidwall/buntdb"
)

// importDatabase is essentially reading a bzip2'ed CSV file with UUID,AvatarName downloaded from http://w-hat.com/#name2key .
//
//	One could theoretically set a cron job to get this file, save it on disk periodically, and keep the database up-to-date
//	see https://stackoverflow.com/questions/24673335/how-do-i-read-a-gzipped-csv-file for the actual usage of these complicated things!
func importDatabase(filename string) {
	filehandler, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer filehandler.Close()

	// First, check if we _do_ have a gzipped file or not...
	// We'll use a small library for that (gwyneth 20211027)

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	_, err = filehandler.Read(head)
	checkErr(err)

	kind, err := filetype.Match(head)
	checkErr(err)
	// Now rewind the file to the start. (gwyneth 20211028)
	position, err := filehandler.Seek(0, 0)
	if position != 0 || err != nil {
		log.Error("could not rewind the file to the start position")
	}

	var cr *csv.Reader // CSV reader needs to be declared here because of scope issues. (gwyneth 20211027)

	// Technically, we could match for a lot of archives and get a io.Reader for each.
	// However, W-Hat has a limited selection of archives available (currently gzip and bzip2)
	// so we limit ourselves to these two, falling back to plaintext (gwyneth 20211027).
	switch kind {
	case matchers.TypeBz2:
		gr := bzip2.NewReader(filehandler) // open bzip2 reader
		cr = csv.NewReader(gr)             // open csv reader and feed the bzip2 reader into it
	case matchers.TypeGz:
		zr, err := gzip.NewReader(filehandler) // open gzip reader
		checkErr(err)
		cr = csv.NewReader(zr) // open csv reader and feed the bzip2 reader into it
	default:
		// We just assume that it's a CSV (uncompressed) file and open it.
		cr = csv.NewReader(filehandler)
	}

	limit := 0               // outside of for loop so that we can count how many entries we had in total
	time_start := time.Now() // we want to get an idea on how long this takes

	switch goslConfig.database {
	case "badger":
		// prepare connection to KV database
		kv, err := badger.Open(Opt)
		checkErrPanic(err) // should probably panic
		defer kv.Close()

		txn := kv.NewTransaction(true) // start new transaction; we will commit only every BATCH_BLOCK entries
		defer txn.Discard()
		for ; ; limit++ {
			record, err := cr.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			// CSV: first entry is avatar key UUID, second entry is avatar name.
			// We probably should check for valid UUIDs; we may do that at some point. (gwyneth 20211031)
			jsonNewEntry, err := json.Marshal(avatarUUID{record[1], record[0], "Production"}) // W-Hat keys come all from the main LL grid, known as 'Production'
			if err != nil {
				log.Warning(err)
			} else {
				log.Debugf("Entry %04d - Name: %s UUID: %s - JSON: %s\n", limit, record[1], record[0], jsonNewEntry)
				// Place this record under the avatar's name
				if err = txn.Set([]byte(record[1]), jsonNewEntry); err != nil {
					log.Fatal(err)
				}
				// Now place it again, this time under the avatar's key
				if err = txn.Set([]byte(record[0]), jsonNewEntry); err != nil {
					log.Fatal(err)
				}
			}
			if limit%goslConfig.BATCH_BLOCK == 0 && limit != 0 { // we do not run on the first time, and then only every BATCH_BLOCK times
				log.Info("processing:", limit)
				if err = txn.Commit(); err != nil {
					log.Fatal(err)
				}
				runtime.GC()
				txn = kv.NewTransaction(true) // start a new transaction
				defer txn.Discard()
			}
		}
		// commit last batch
		if err = txn.Commit(); err != nil {
			log.Fatal(err)
		}
	case "buntdb":
		db, err := buntdb.Open(goslConfig.dbNamePath)
		checkErrPanic(err)
		defer db.Close()

		txn, err := db.Begin(true)
		checkErrPanic(err)
		//defer txn.Commit()

		// very similar to Badger code...
		for ; ; limit++ {
			record, err := cr.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			jsonNewEntry, err := json.Marshal(avatarUUID{record[1], record[0], "Production"})
			if err != nil {
				log.Warning(err)
			} else {
				// see comments above for Badger. (gwyneth 20211031)
				_, _, err = txn.Set(record[1], string(jsonNewEntry), nil)
				if err != nil {
					log.Fatal(err)
				}
				_, _, err = txn.Set(record[0], string(jsonNewEntry), nil)
				if err != nil {
					log.Fatal(err)
				}
			}
			if limit%goslConfig.BATCH_BLOCK == 0 && limit != 0 { // we do not run on the first time, and then only every BATCH_BLOCK times
				log.Info("processing:", limit)
				if err = txn.Commit(); err != nil {
					log.Fatal(err)
				}
				runtime.GC()
				txn, err = db.Begin(true) // start a new transaction
				checkErrPanic(err)
				//defer txn.Commit()
			}
		}
		// commit last batch
		if err = txn.Commit(); err != nil {
			log.Fatal(err)
		}
		db.Shrink()
	case "leveldb":
		db, err := leveldb.OpenFile(goslConfig.dbNamePath, nil)
		checkErrPanic(err)
		defer db.Close()
		batch := new(leveldb.Batch)

		for ; ; limit++ {
			record, err := cr.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			jsonNewEntry, err := json.Marshal(avatarUUID{record[1], record[0], "Production"})
			if err != nil {
				log.Warning(err)
			} else {
				// see comments above for Badger. (gwyneth 20211031)
				batch.Put([]byte(record[1]), jsonNewEntry)
				batch.Put([]byte(record[0]), jsonNewEntry)
			}
			if limit%goslConfig.BATCH_BLOCK == 0 && limit != 0 {
				log.Info("processing:", limit)
				if err = db.Write(batch, nil); err != nil {
					log.Fatal(err)
				}
				batch.Reset() // unlike the others, we don't need to create a new batch every time
				runtime.GC()  // it never hurts...
			}
		}
		// commit last batch
		if err = db.Write(batch, nil); err != nil {
			log.Fatal(err)
		}
		batch.Reset() // reset it and let the garbage collector run
		runtime.GC()
		db.CompactRange(util.Range{Start: nil, Limit: nil})
	}
	log.Info("total read", limit, "records (or thereabouts) in", time.Since(time_start))
}
