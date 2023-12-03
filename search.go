// This file will just have the search functions.
// Note that the deprecated versions are still kept around, just in case I need to figure out again
// how iterators work...
// (gwyneth 20211102)
package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/buntdb"
)

// searchKVname searches the KV database for an avatar name.
// Returns NullUUID if the key wasn't found.
func searchKVname(avatarName string) (uuid string, grid string) {
	_, tempUUID, tempGrid := searchKV(strings.TrimSpace(avatarName))
	return tempUUID, tempGrid
}

// searchKVUUID searches the KV database for an avatar UUID.
// Returns empty string if the avatar name wasn't found.
func searchKVUUID(avatarKey string) (name string, grid string) {
	tempAvatarName, _, tempGrid := searchKV(strings.TrimSpace(avatarKey))
	return tempAvatarName, tempGrid
}

// Universal search, since we put everything in the KV database, we can basically search for anything.
// *Way* more efficient! (gwyneth 20211031)
// Returns the unmarshaled record from the KV store, if found;
// otherwise, avatar name will be the empty string and avatar key will be NullUUID.
func searchKV(searchItem string) (name string, uuid string, grid string) {
	// return value.
	var val = avatarUUID{"", NullUUID, ""}
	var err error // to deal with scope issues.
	time_start := time.Now()	// start chroometer to time this transaction.
	switch goslConfig.database {
	case "badger":
		kv, err = badger.Open(Opt)
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
				log.Errorf("error while unmarshalling reply to search item: %q (%v)\n", searchItem, err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Errorf("error while getting or unmarshalling reply to search item: %q (%v)\n", searchItem, err)
		}
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
			log.Errorf("error while unmarshalling reply to search item: %q (%v)\n", searchItem, err)
		}
	case "leveldb":
		db, errdb := leveldb.OpenFile(goslConfig.dbNamePath, nil)
		checkErrPanic(errdb)
		defer db.Close()
		var data []byte	// for scoping reasons.
		data, err = db.Get([]byte(searchItem), nil)
		if err != nil {
			log.Errorf("error while unmarshalling reply to search item: %q (%v)\n", searchItem, err)
		} else {
			if err = json.Unmarshal(data, &val); err != nil {
				log.Errorf("error while unmarshalling reply to search item: %q (%v)\n", searchItem, err)
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
