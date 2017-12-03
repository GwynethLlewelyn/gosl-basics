// gosl is a basic example of how to develop external web services for Second Life/OpenSimulator using the Go programming language.
package main

import (
	"bufio"
	"compress/bzip2"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/fsnotify/fsnotify"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/tidwall/buntdb"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"net/http"
	"net/http/fcgi"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const NullUUID = "00000000-0000-0000-0000-000000000000" // always useful when we deal with SL/OpenSimulator...
const databaseName = "gosl-database.db" // for BuntDB

// Logging setup.
var log = logging.MustGetLogger("gosl")	// configuration for the go-logging logger, must be available everywhere
var logFormat logging.Formatter

// Opt is used for Badger database setup.
var Opt badger.Options

// AvatarUUID is the type that we store in the database; we keep a record from which grid it came from.
type avatarUUID struct {
	UUID string		// needs to be capitalised for JSON marshalling (it has to do with the way it works)
	Grid string
} 

/*
				  .__			 
  _____ _____  |__| ____  
 /		  \\__	 \ |	 |/	\ 
|  Y Y	\/ __ \|	 |		 |	 \
|__|_|	(____  /__|___|	 /
	  \/	 \/			  \/ 
*/

// Configuration options
type goslConfigOptions struct {
	BATCH_BLOCK int // how many entries to write to the database as a block; the bigger, the faster, but the more memory it consumes
	noMemory, isServer, isShell *bool
	myDir, myPort, importFilename, database *string
	dbNamePath string // for BuntDB
	logFilename string	// for logs
	maxSize, maxBackups, maxAge int // logs configuration option
}

var goslConfig goslConfigOptions

// loadConfiguration reads our configuration from a config.toml file
func loadConfiguration() {
	fmt.Print("Reading gosl-basic configuration:")	// note that we might not have go-logging active as yet, so we use fmt
	// Open our config file and extract relevant data from there
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return	// we might still get away with this!
	}
	viper.SetDefault("config.BATCH_BLOCK", 100000)	// NOTE(gwyneth): the authors of say that 100000 is way too much for Badger																// NOTE(gwyneth): let's see what happens with BuntDB
	goslConfig.BATCH_BLOCK = viper.GetInt("config.BATCH_BLOCK")
	viper.SetDefault("config.myPort", 3000)
	*goslConfig.myPort = viper.GetString("config.myPort")
	viper.SetDefault("config.myDir", "slkvdb")
	*goslConfig.myDir = viper.GetString("config.myDir")
	viper.SetDefault("config.isServer", false)
	*goslConfig.isServer = viper.GetBool("config.isServer")
	viper.SetDefault("config.isShell", false)
	*goslConfig.isShell = viper.GetBool("config.isShell")
	viper.SetDefault("config.database", "badger")
	*goslConfig.database = viper.GetString("config.database")
	viper.SetDefault("config.importFilename", "") // must be empty by default
	*goslConfig.importFilename = viper.GetString("config.importFilename")
	viper.SetDefault("config.noMemory", false)
	*goslConfig.noMemory = viper.GetBool("config.noMemory")
	// Logging options
	viper.SetDefault("config.logFilename", "gosl.log")
	goslConfig.logFilename = viper.GetString("config.logFilename")
	viper.SetDefault("config.maxSize", 10)
	goslConfig.maxSize = viper.GetInt("config.maxSize")
	viper.SetDefault("config.maxBackups", 3)
	goslConfig.maxBackups = viper.GetInt("config.maxBackups")
	viper.SetDefault("config.maxAge", 28)
	goslConfig.maxAge = viper.GetInt("config.maxAge")
}

// main() starts here.
func main() {
	// Flag setup; can be overridden by config file (I need to fix this to be the oher way round).
	goslConfig.myPort			= flag.String("port", "3000", "Server port")
	goslConfig.myDir			= flag.String("dir", "slkvdb", "Directory where database files are stored")
	goslConfig.isServer			= flag.Bool("server", false, "Run as server on port " + *goslConfig.myPort)
	goslConfig.isShell			= flag.Bool("shell", false, "Run as an interactive shell")
	goslConfig.importFilename	= flag.String("import", "name2key.csv.bz2", "Import database from W-Hat (use the csv.bz2 version)")
	goslConfig.database 		= flag.String("database", "badger", "Database type (currently BuntDB or Badger)")
	goslConfig.noMemory 		= flag.Bool("nomemory", false, "Attempt to use only disk to save memory on Badger (important for shared webservers)")
	
	// Config viper, which reads in the configuration file every time it's needed.
	// Note that we need some hard-coded variables for the path and config file name.
	viper.SetConfigName("config")
	viper.SetConfigType("toml") // just to make sure; it's the same format as OpenSimulator (or MySQL) config files
	viper.AddConfigPath("$HOME/go/src/gosl-basics/") // that's how I have it
	viper.AddConfigPath("$HOME/go/src/git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics/") // that's how you'll have it
	viper.AddConfigPath(".")               // optionally look for config in the working directory
	
	loadConfiguration()

	// this will allow our configuration file to be 'read on demand'
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		if *goslConfig.isServer || *goslConfig.isShell {
			fmt.Println("Config file changed:", e.Name)	// BUG(gwyneth): FastCGI cannot write to output
		}
		loadConfiguration()
	})
		
	// default is FastCGI
	flag.Parse()
	
	// NOTE(gwyneth): We cannot write to stdout if we're running as FastCGI, only to logs!
	if *goslConfig.isServer || *goslConfig.isShell {
		fmt.Println("gosl is starting...")	
	}
	
	// Setup the lumberjack rotating logger. This is because we need it for the go-logging logger when writing to files. (20170813)
	rotatingLogger := &lumberjack.Logger{
		Filename:	goslConfig.logFilename,
		MaxSize:	goslConfig.maxSize, // megabytes
		MaxBackups:	goslConfig.maxBackups,
		MaxAge:		goslConfig.maxAge, //days
	}
	
	// Set formatting for stderr and file (basically the same).
	logFormat := logging.MustStringFormatter(`%{color}%{time:2006/01/02 15:04:05.0} %{shortfile} - %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`) 	// must be initialised or all hell breaks loose
	
	// Setup the go-logging Logger. Do **not** log to stderr if running as FastCGI!
	backendFile				:= logging.NewLogBackend(rotatingLogger, "", 0)
	backendFileFormatter	:= logging.NewBackendFormatter(backendFile, logFormat)
	backendFileLeveled 		:= logging.AddModuleLevel(backendFileFormatter)
	backendFileLeveled.SetLevel(logging.INFO, "gosl")	// we just send debug data to logs if we run as shell
	
	if *goslConfig.isServer || *goslConfig.isShell {
		backendStderr			:= logging.NewLogBackend(os.Stderr, "", 0)
		backendStderrFormatter	:= logging.NewBackendFormatter(backendStderr, logFormat)
		backendStderrLeveled 	:= logging.AddModuleLevel(backendStderrFormatter)
		if *goslConfig.isShell {
			backendStderrLeveled.SetLevel(logging.DEBUG, "gosl")	// shell is meant to be for debugging mostly
		} else {
			backendStderrLeveled.SetLevel(logging.INFO, "gosl")
		}
		logging.SetBackend(backendStderrLeveled, backendFileLeveled)
	} else {
		logging.SetBackend(backendFileLeveled)	// FastCGI only logs to file
	}

	// Check if this directory actually exists; if not, create it. Panic if something wrong happens (we cannot proceed without a valid directory for the database to be written
	if stat, err := os.Stat(*goslConfig.myDir); err == nil && stat.IsDir() {
		// path is a valid directory
		log.Infof("Valid directory: %s\n", *goslConfig.myDir)
	} else {
		// try to create directory
		err = os.Mkdir(*goslConfig.myDir, 0700)
		checkErrPanic(err) // cannot make directory, panic and exit logging what went wrong
		log.Debugf("Created new directory: %s\n", *goslConfig.myDir)		
	}
	if *goslConfig.database == "badger" {
		Opt = badger.DefaultOptions
		Opt.Dir = *goslConfig.myDir
		Opt.ValueDir = Opt.Dir
		Opt.TableLoadingMode = options.MemoryMap
		//Opt.TableLoadingMode = options.FileIO
	
		if *goslConfig.noMemory  {
	//		Opt.TableLoadingMode = options.FileIO // use standard file I/O operations for tables instead of LoadRAM
	//		Opt.TableLoadingMode = options.MemoryMap // MemoryMap indicates that that the file must be memory-mapped - https://github.com/dgraph-io/badger/issues/224#issuecomment-329643771
			Opt.TableLoadingMode = options.FileIO
	//		Opt.ValueLogFileSize = 1048576
			Opt.MaxTableSize = 1048576 // * 12
			Opt.LevelSizeMultiplier = 1
			Opt.NumMemtables = 1
	//		Opt.MaxLevels = 10
	//		Opt.SyncWrites = false
	//		Opt.NumCompactors = 10
	//		Opt.NumLevelZeroTables = 10
	//		Opt.maxBatchSize =
	//		Opt.maxBatchCount =
			goslConfig.BATCH_BLOCK = 1000	// try to import less at each time, it will take longer but hopefully work
			log.Info("Trying to avoid too much memory consumption")	
		}
	}
	// Do some testing to see if the database is available				
	const testAvatarName = "Nobody Here"
	var err error

	log.Info("gosl started and logging is set up. Proceeding to test database (" + *goslConfig.database + ") at " + *goslConfig.myDir)
	var testValue = avatarUUID{ NullUUID, "all grids" }
	jsonTestValue, err := json.Marshal(testValue)
	checkErrPanic(err) // something went VERY wrong

	if *goslConfig.database == "badger" {
		kv, err := badger.Open(Opt)
		checkErrPanic(err) // should probably panic, cannot prep new database
		txn := kv.NewTransaction(true)
		err = txn.Set([]byte(testAvatarName), jsonTestValue)
		checkErrPanic(err)
		err = txn.Commit(nil)
		checkErrPanic(err)
		log.Debugf("SET %+v (json: %v)\n", testValue, string(jsonTestValue))
		kv.Close()
	} else if *goslConfig.database == "buntdb" {
		/* NOTE(gwyneth): this fails because pointers to strings do not implement len(). Duh! 
		if *goslConfig.myDir[len(*goslConfig.myDir)-1] != os.PathSeparator {
			*goslConfig.myDir = append(*goslConfig.myDir + os.PathSeparator
		} */
		goslConfig.dbNamePath = *goslConfig.myDir + string(os.PathSeparator) + databaseName
		db, err := buntdb.Open(goslConfig.dbNamePath)
		checkErrPanic(err)
		err = db.Update(func(tx *buntdb.Tx) error {
			_, _, err := tx.Set(testAvatarName, string(jsonTestValue), nil)
			return err
		})
		checkErr(err)
		log.Debugf("SET %+v (json: %v)\n", testValue, string(jsonTestValue))
		db.Close()
	}
	// common to both databases:
	key, grid := searchKVname(testAvatarName)
	log.Debugf("GET '%s' returned '%s' [grid '%s']\n", testAvatarName, key, grid)
	log.Info("KV database seems fine.")
	
	if *goslConfig.importFilename != "" {
		log.Info("Attempting to import", *goslConfig.importFilename, "...")
		importDatabase(*goslConfig.importFilename)
		log.Info("Database finished import.")
	}
	
	if *goslConfig.isShell {
		log.Info("Starting to run as interactive shell")
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Ctrl-C to quit.")
		var err error	// to avoid assigning text in a different scope (this is a bit awkward, but that's the problem with bi-assignment)
		var checkInput, avatarName, avatarKey, gridName string
		for {
			// Prompt and read			
			fmt.Print("Enter avatar name or UUID: ")
			checkInput, err = reader.ReadString('\n')
			checkErr(err)
			checkInput = strings.TrimRight(checkInput, "\r\n")
			// fmt.Printf("Ok, got %s length is %d and UUID is %v\n", checkInput, len(checkInput), isValidUUID(checkInput))
			if (len(checkInput) == 36) && isValidUUID(checkInput) {
				avatarName, gridName = searchKVUUID(checkInput)
				avatarKey = checkInput
			} else {				
				avatarKey, gridName = searchKVname(checkInput)
				avatarName = checkInput
			}
			if avatarName != NullUUID && avatarKey != NullUUID {
				fmt.Println(avatarName, "which has UUID:", avatarKey, "comes from grid:", gridName)	
			} else {
				fmt.Println("Sorry, unknown input", checkInput)
			}	
		}
		// never leaves until Ctrl-C
	}
	
	// set up routing.
	// NOTE(gwyneth): one function only because FastCGI seems to have problems with multiple handlers.
	http.HandleFunc("/", handler)
	log.Info("Directory for database:", *goslConfig.myDir)
	
	if (*goslConfig.isServer) {
		log.Info("Starting to run as web server on port " + *goslConfig.myPort)
		err := http.ListenAndServe(":" + *goslConfig.myPort, nil) // set listen port
		checkErrPanic(err) // if it can't listen to all the above, then it has to abort anyway
	} else {
		// default is to run as FastCGI!
		// works like a charm thanks to http://www.dav-muz.net/blog/2013/09/how-to-use-go-and-fastcgi/
		log.Debug("http.DefaultServeMux is", http.DefaultServeMux)
		if err := fcgi.Serve(nil, nil); err != nil {
			checkErrPanic(err)
		}
	}
	// we should never have reached this point!
	log.Error("Unknown usage! This application may run as a standalone server, as FastCGI application, or as an interactive shell")
	if *goslConfig.isServer || *goslConfig.isShell {
		flag.PrintDefaults()
	}	
}

// handler deals with incoming queries and/or associates avatar names with keys depending on parameters.
// Basically we check if both an avatar name and a UUID key has been received: if yes, this means a new entry;
//	if just the avatar name was received, it means looking up its key;
//	if just the key was received, it means looking up the name (not necessary since llKey2Name does that, but it's just to illustrate);
//	if nothing is received, then return an error
func handler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logErrHTTP(w, http.StatusNotFound, "No avatar and/or UUID received")
		return
	}
	// test first if this comes from Second Life or OpenSimulator
/*
	if r.Header.Get("X-Secondlife-Region") == "" {
		logErrHTTP(w, http.StatusForbidden, "Sorry, this application only works inside Second Life.")
		return
	}
*/
	name := r.Form.Get("name") // can be empty
	key := r.Form.Get("key") // can be empty
	compat := r.Form.Get("compat") // compatibility mode with W-Hat
	var valueToInsert avatarUUID
	messageToSL := "" // this is what we send back to SL - defined here due to scope issues.
	if name != "" {
		if key != "" {
			// we received both: add a new entry
			valueToInsert.UUID = key
			valueToInsert.Grid = r.Header.Get("X-Secondlife-Shard")
			jsonValueToInsert, err := json.Marshal(valueToInsert)
			checkErr(err)
			if *goslConfig.database == "badger" {
				kv, err := badger.Open(Opt)
				checkErrPanic(err) // should probably panic
				txn := kv.NewTransaction(true)
				defer txn.Discard()
				err = txn.Set([]byte(name), jsonValueToInsert)
				checkErrPanic(err)
				err = txn.Commit(nil)
				checkErrPanic(err)
				kv.Close()
			} else if *goslConfig.database == "buntdb" {
				db, err := buntdb.Open(goslConfig.dbNamePath)
				checkErrPanic(err)
				defer db.Close()				
				err = db.Update(func(tx *buntdb.Tx) error {
					_, _, err := tx.Set(name, string(jsonValueToInsert), nil)
					return err
				})
				checkErr(err)
			}
			messageToSL += "Added new entry for '" + name + "' which is: " + valueToInsert.UUID + " from grid: '" + valueToInsert.Grid + "'"
		} else {
			// we received a name: look up its UUID key and grid.
			key, grid := searchKVname(name)
			if compat == "false" {
				messageToSL += "UUID for '" + name + "' is: " + key + " from grid: '" + grid + "'"
			} else { // empty also means true!
				messageToSL += key		
			}
		}
	} else if key != "" {
		// in this scenario, we have the UUID key but no avatar name: do the equivalent of a llKey2Name (slow)
		name, grid := searchKVUUID(key)
		if compat == "false" {
			messageToSL += "Avatar name for " + key + "' is '" + name + "' on grid: '" + grid + "'"
		} else { // empty also means true!
			messageToSL += name		
		}
	} else {
		// neither UUID key nor avatar received, this is an error
		logErrHTTP(w, http.StatusNotFound, "Empty avatar name and UUID key received, cannot proceed")
		return	
	}	
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, messageToSL)
}
// searchKVname searches the KV database for an avatar name.
func searchKVname(avatarName string) (UUID string, grid string) {
	var val = avatarUUID{ NullUUID, "" }
	time_start := time.Now()
	var err error // to deal with scope issues
	if *goslConfig.database == "badger" {
		kv, err := badger.Open(Opt)
		checkErrPanic(err)
		defer kv.Close()
		err = kv.View(func(txn *badger.Txn) error {
		    item, err := txn.Get([]byte(avatarName))
			if err != nil {
				return err
	    	}    	
	    	data, err := item.Value()
			if err != nil {
				log.Errorf("Error '%s' while getting data from %v\n", err, item)
				return err
	    	}    	
	    	err = json.Unmarshal(data, &val)
			if err != nil {
				log.Errorf("Error while unparsing UUID for name: '%s' (%v)\n", avatarName, err)
				return err
	    	}
	    	return nil
		})	
	} else if *goslConfig.database == "buntdb" {
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
			log.Errorf("Error while unparsing UUID for name: '%s' (%v)\n", avatarName, err)
    	}			
	}
	time_end := time.Now()
	diffTime := time_end.Sub(time_start)
	log.Debugf("Time to lookup '%s': %v\n", avatarName, diffTime)
	if err != nil {
		return NullUUID, ""
	} // else:
	return val.UUID, val.Grid
}
// searchKVUUID searches the KV database for an avatar key.
func searchKVUUID(avatarKey string) (name string, grid string) {
	time_start := time.Now()
	checks := 0
	var val = avatarUUID{ NullUUID, "" }
	var found string
	
	if *goslConfig.database == "badger" {
		kv, err := badger.Open(Opt)
		checkErr(err) // should probably panic
		itOpt := badger.DefaultIteratorOptions
/*
		if !*goslConfig.noMemory {
			itOpt.PrefetchValues = true
			itOpt.PrefetchSize = 1000	// attempt to get this a little bit more efficient; we have many small entries, so this is not too much
		} else {
*/
			itOpt.PrefetchValues = false // allegedly this is supposed to be WAY faster...
// 		}
		txn := kv.NewTransaction(true)
		defer txn.Discard()
	
		err = kv.View(func(txn *badger.Txn) error {				
			itr := txn.NewIterator(itOpt)
			defer itr.Close()		
			for itr.Rewind(); itr.Valid(); itr.Next() {
				item := itr.Item()
				data, err := item.Value()
				if err != nil {
					log.Errorf("Error '%s' while getting data from %v\n", err, item)
					return err
		    	}    	
		    	err = json.Unmarshal(data, &val)
				if err != nil {
					log.Errorf("Error '%s' while unparsing UUID for data: %v\n", err, data)
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
		kv.Close()
	} else {
		db, err := buntdb.Open(goslConfig.dbNamePath)
		checkErrPanic(err)
		err = db.View(func(tx *buntdb.Tx) error {
			err := tx.Ascend("", func(key, value string) bool {
		    	err = json.Unmarshal([]byte(value), &val)
				if err != nil {
					log.Errorf("Error '%s' while unparsing UUID for value: %v\n", err, value)
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
	}
	time_end := time.Now()
	diffTime := time_end.Sub(time_start)
	log.Debugf("Made %d checks for '%s' in %v\n", checks, avatarKey, diffTime)
	return found, val.Grid
}

// importDatabase is essentially reading a bzip2'ed CSV file with UUID,AvatarName downloaded from http://w-hat.com/#name2key .
//	One could theoretically set a cron job to get this file, save it on disk periodically, and keep the database up-to-date
//	see https://stackoverflow.com/questions/24673335/how-do-i-read-a-gzipped-csv-file for the actual usage of these complicated things!
func importDatabase(filename string) {
	filehandler, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer filehandler.Close()
	gr := bzip2.NewReader(filehandler) // open bzip2 reader
	cr := csv.NewReader(gr)  // open csv reader and feed the bzip2 reader into it

	limit := 0	// outside of for loop so that we can count how many entries we had in total
	time_start := time.Now() // we want to get an idea on how long this takes
	
	if *goslConfig.database == "badger" {
		// prepare connection to KV database
		kv, err := badger.Open(Opt)
		checkErrPanic(err) // should probably panic		
		defer kv.Close()	
	
		txn := kv.NewTransaction(true) // start new transaction; we will commit only every BATCH_BLOCK entries
		defer txn.Discard()
		for ;;limit++ {
			record, err := cr.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			jsonNewEntry, err := json.Marshal(avatarUUID{ record[0], "Production" }) // W-Hat keys come all from the main LL grid, known as 'Production'
			if err != nil {
				log.Warning(err)
			} else {			 
				err = txn.Set([]byte(record[1]), jsonNewEntry)
				if err != nil {
				    log.Fatal(err)
				}
			}
			if limit % goslConfig.BATCH_BLOCK == 0 && limit != 0 { // we do not run on the first time, and then only every BATCH_BLOCK times
				log.Info("Processing:", limit)
				err = txn.Commit(nil)
				if err != nil {
				    log.Fatal(err)
				}
				runtime.GC()
				txn = kv.NewTransaction(true) // start a new transaction
				defer txn.Discard()
			}
		}
		// commit last batch
		err = txn.Commit(nil)
		if err != nil {
		    log.Fatal(err)
		}
		kv.PurgeOlderVersions()
	} else {
		db, err := buntdb.Open(goslConfig.dbNamePath)
		checkErrPanic(err)
		defer db.Close()
		
		txn, err := db.Begin(true)
		checkErrPanic(err)
		//defer txn.Commit()
		
		// very similar to Badger code...
		for ;;limit++ {
			record, err := cr.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			jsonNewEntry, err := json.Marshal(avatarUUID{ record[0], "Production" }) // W-Hat keys come all from the main LL grid, known as 'Production'
			if err != nil {
				log.Warning(err)
			} else {			 
				_, _, err = txn.Set(record[1], string(jsonNewEntry), nil)
				if err != nil {
				    log.Fatal(err)
				}
			}
			if limit % goslConfig.BATCH_BLOCK == 0 && limit != 0 { // we do not run on the first time, and then only every BATCH_BLOCK times
				log.Info("Processing:", limit)
				err = txn.Commit()
				if err != nil {
				    log.Fatal(err)
				}
				runtime.GC()
				txn, err = db.Begin(true)  // start a new transaction
				checkErrPanic(err)
				//defer txn.Commit()
			}
		}
		// commit last batch
		err = txn.Commit()
		if err != nil {
		    log.Fatal(err)
		}			
		db.Shrink()
	}
	time_end := time.Now()
	diffTime := time_end.Sub(time_start)
	log.Info("Total read", limit, "records (or thereabouts) in", diffTime)
}

// NOTE(gwyneth): Auxiliary functions which I'm always using...

// checkErrPanic logs a fatal error and panics.
func checkErrPanic(err error) {
	if err != nil {
		pc, file, line, ok := runtime.Caller(1)
		log.Panic(filepath.Base(file), ":", line, ":", pc, ok, " - panic:", err)
	}
}
// checkErr checks if there is an error, and if yes, it logs it out and continues.
//	this is for 'normal' situations when we want to get a log if something goes wrong but do not need to panic
func checkErr(err error) {
	if err != nil {
		pc, file, line, ok := runtime.Caller(1)
		log.Error(filepath.Base(file), ":", line, ":", pc, ok, " - error:", err)
	}
}

// Auxiliary functions for HTTP handling

// checkErrHTTP returns an error via HTTP and also logs the error.
func checkErrHTTP(w http.ResponseWriter, httpStatus int, errorMessage string, err error) {
	if err != nil {
		http.Error(w, fmt.Sprintf(errorMessage, err), httpStatus)
		pc, file, line, ok := runtime.Caller(1)
		log.Error("(", http.StatusText(httpStatus), ") ", filepath.Base(file), ":", line, ":", pc, ok, " - error:", errorMessage, err)	
	}
}
// checkErrPanicHTTP returns an error via HTTP and logs the error with a panic.
func checkErrPanicHTTP(w http.ResponseWriter, httpStatus int, errorMessage string, err error) {
	if err != nil {
		http.Error(w, fmt.Sprintf(errorMessage, err), httpStatus)
		pc, file, line, ok := runtime.Caller(1)
		log.Panic("(", http.StatusText(httpStatus), ") ", filepath.Base(file), ":", line, ":", pc, ok, " - panic:", errorMessage, err)
	}
}
// logErrHTTP assumes that the error message was already composed and writes it to HTTP and logs it.
//	this is mostly to avoid code duplication and make sure that all entries are written similarly 
func logErrHTTP(w http.ResponseWriter, httpStatus int, errorMessage string) {
	http.Error(w, errorMessage, httpStatus)
	log.Error("(" + http.StatusText(httpStatus) + ") " + errorMessage)
}
// funcName is @Sonia's solution to get the name of the function that Go is currently running.
//	This will be extensively used to deal with figuring out where in the code the errors are!
//	Source: https://stackoverflow.com/a/10743805/1035977 (20170708)
func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}

// isValidUUID checks if the UUID is valid.
// Thanks to Patrick D'Appollonio https://stackoverflow.com/questions/25051675/how-to-validate-uuid-v4-in-go
func isValidUUID(uuid string) bool {
    r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
    return r.MatchString(uuid)
}