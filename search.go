// This file will just have the search functions.
// Note that the deprecated versions are still kept around, just in case I need to figure out again
// how iterators work...
// (gwyneth 20211102)
package main

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/buntdb"
)

// searchKVname searches the KV database for an avatar name.
func searchKVname(avatarName string) (uuid string, grid string) {
	_, tempUUID, tempGrid := searchKV(avatarName)
	return tempUUID, tempGrid
}

// searchKVname searches the KV database for an avatar name.
func searchKVUUID(avatarKey string) (name string, grid string) {
	tempAvatarName, _, tempGrid := searchKV(avatarKey)
	return tempAvatarName, tempGrid
}

/* deprecated (gwyneth 20211031)

// searchKVname searches the KV database for an avatar name.
func searchKVname(avatarName string) (UUID string, grid string) {
	var val = avatarUUID{ avatarName, NullUUID, "" }
	time_start := time.Now()
	var err error // to deal with scope issues
	switch *goslConfig.database {
		case "badger":
			kv, err := badger.Open(Opt)
			checkErrPanic(err)
			defer kv.Close()
			err = kv.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte(avatarName))
				if err != nil {
					return err
				}
				data, err := item.ValueCopy(nil)
				if err != nil {
					log.Errorf("error %q while getting data from %v\n", err, item)
					return err
				}
				if err = json.Unmarshal(data, &val); err != nil {
					log.Errorf("error while unparsing UUID for name: %q (%v)\n", avatarName, err)
					return err
				}
				return nil
			})
			checkErr(err)
		case "buntdb":
			db, err := buntdb.Open(goslConfig.dbNamePath)
			checkErrPanic(err)
			defer db.Close()
			var data string
			err = db.View(func(tx *buntdb.Tx) error {
				data, err = tx.Get(avatarName)
				return err
			})
			err = json.Unmarshal([]byte(data), &val)
			if err != nil {
				log.Errorf("error while unparsing UUID for name: %q (%v)\n", avatarName, err)
			}
		case "leveldb":
			db, err := leveldb.OpenFile(goslConfig.dbNamePath, nil)
			checkErrPanic(err)
			defer db.Close()
			data, err := db.Get([]byte(avatarName), nil)
			if err != nil {
				log.Errorf("error while getting UUID for name: %q (%v)\n", avatarName, err)
			} else {
				if err = json.Unmarshal(data, &val); err != nil {
					log.Errorf("error while unparsing UUID for name: %q (%v)\n", avatarName, err)
				}
			}
	}
	log.Debugf("time to lookup %q: %v\n", avatarName, time.Since(time_start))
	if err != nil {
		return NullUUID, ""
	} // else:
	return val.UUID, val.Grid
}
// searchKVUUID searches the KV database for an avatar key.
func searchKVUUID(avatarKey string) (name string, grid string) {
	time_start := time.Now()
	checks := 0
	var val = avatarUUID{ "", avatarKey, "" }
	var found string

	switch *goslConfig.database {
		case "badger":
			kv, err := badger.Open(Opt)
			checkErr(err) // should probably panic
			itOpt := badger.DefaultIteratorOptions
	//
	//		if !*goslConfig.noMemory {
	//			itOpt.PrefetchValues = true
	//			itOpt.PrefetchSize = 1000	// attempt to get this a little bit more efficient; we have many small entries, so this is not too much
	//		} else {
	//
				itOpt.PrefetchValues = false // allegedly this is supposed to be WAY faster...
	// 		}
			txn := kv.NewTransaction(true)
			defer txn.Discard()

			err = kv.View(func(txn *badger.Txn) error {
				itr := txn.NewIterator(itOpt)
				defer itr.Close()
				for itr.Rewind(); itr.Valid(); itr.Next() {
					item := itr.Item()
					data, err := item.ValueCopy(nil)
					if err != nil {
						log.Errorf("error %q while getting data from %v\n", err, item)
						return err
					}
					if err = json.Unmarshal(data, &val); err != nil {
						log.Errorf("error %q while unparsing UUID for data: %v\n", err, data)
						return err
					}
					checks++	//Just to see how many checks we made, for statistical purposes
					if avatarKey == val.UUID {
						found = string(item.Key())
						break
					}
				}
				return nil
			})
			checkErr(err)
			kv.Close()
		case "buntdb":
			db, err := buntdb.Open(goslConfig.dbNamePath)
			checkErrPanic(err)
			err = db.View(func(tx *buntdb.Tx) error {
				err := tx.Ascend("", func(key, value string) bool {
					if err = json.Unmarshal([]byte(value), &val); err != nil {
						log.Errorf("error %q while unparsing UUID for value: %v\n", err, value)
					}
					checks++	//Just to see how many checks we made, for statistical purposes
					if avatarKey == val.UUID {
						found = key
						return false
					}
					return true
				})
				return err
			})
			db.Close()
		case "leveldb":
			db, err := leveldb.OpenFile(goslConfig.dbNamePath, nil)
			checkErrPanic(err)
			iter := db.NewIterator(nil, nil)
			for iter.Next() {
				// Remember that the contents of the returned slice should not be modified, and
				// only valid until the next call to Next.
				key := iter.Key()
				value := iter.Value()
				if err = json.Unmarshal(value, &val); err != nil {
					log.Errorf("error %q while unparsing UUID for data: %v\n", err, value)
					continue // a bit insane, but at least we will skip a few broken records
				}
				checks++	//Just to see how many checks we made, for statistical purposes
				if avatarKey == val.UUID {
					found = string(key)
					break
				}
			}
			iter.Release()
			err = iter.Error()
			checkErr(err)
			db.Close()
	} // /switch
	log.Debugf("made %d checks for %q in %v\n", checks, avatarKey, time.Since(time_start))
	return found, val.Grid
}
*/

// Universal search, since we put everything in the KV database, we can basically search for anything.
// *Way* more efficient! (gwyneth 20211031)
func searchKV(searchItem string) (name string, uuid string, grid string) {
	var val = avatarUUID{ "", NullUUID, "" }
	time_start := time.Now()
	var err error // to deal with scope issues
	switch goslConfig.database {
		case "badger":
			kv, err := badger.Open(Opt)
			checkErrPanic(err)
			defer kv.Close()
			err = kv.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte(searchItem))
				if err != nil {
					return err
				}
				data, err := item.ValueCopy(nil)
				if err != nil {
					log.Errorf("error %q while getting data from %v\n", err, item)
					return err
				}
				if err = json.Unmarshal(data, &val); err != nil {
					log.Errorf("error while unparsing UUID for name: %q (%v)\n", searchItem, err)
					return err
				}
				return nil
			})
			checkErr(err)
		case "buntdb":
			db, err := buntdb.Open(goslConfig.dbNamePath)
			checkErrPanic(err)
			defer db.Close()
			var data string
			err = db.View(func(tx *buntdb.Tx) error {
				data, err = tx.Get(searchItem)
				return err
			})
			err = json.Unmarshal([]byte(data), &val)
			if err != nil {
				log.Errorf("error while unparsing UUID for name: %q (%v)\n", searchItem, err)
			}
		case "leveldb":
			db, err := leveldb.OpenFile(goslConfig.dbNamePath, nil)
			checkErrPanic(err)
			defer db.Close()
			data, err := db.Get([]byte(searchItem), nil)
			if err != nil {
				log.Errorf("error while getting UUID for name: %q (%v)\n", searchItem, err)
			} else {
				if err = json.Unmarshal(data, &val); err != nil {
					log.Errorf("error while unparsing UUID for name: %q (%v)\n", searchItem, err)
				}
			}
	}
	log.Debugf("time to lookup %q: %v\n", searchItem, time.Since(time_start))
	if err != nil {
		checkErr(err)
		return "", NullUUID, ""
	} // else:
	return val.AvatarName, val.UUID, val.Grid
}