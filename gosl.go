// gosl implements the name2key/key2name functionality for about
// ten million avatar names (1/6 of the total database)
package main

import (
	//	"bufio"			// replaced by the more sophisticated readline (gwyneth 20211106)
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/fcgi"
	"os"
	"path/filepath"
	//	"regexp"
	"strings"
	//	"time"

	"github.com/dgraph-io/badger/v3"
	//	"github.com/dgraph-io/badger/options"
	//	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/op/go-logging"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/buntdb"
	"gitlab.com/cznic/readline"
	//	"gopkg.in/go-playground/validator.v9"	// to validate UUIDs... and a lot of thinks
	"gopkg.in/natefinch/lumberjack.v2"
)

const NullUUID = "00000000-0000-0000-0000-000000000000" // always useful when we deal with SL/OpenSimulator...
const databaseName = "gosl-database.db"                 // for BuntDB

// Logging setup.
var log = logging.MustGetLogger("gosl") // configuration for the go-logging logger, must be available everywhere
var logFormat logging.Formatter

// Opt is used for Badger database setup.
var Opt badger.Options

// AvatarUUID is the type that we store in the database; we keep a record from which grid it came from.
// Field names need to be capitalised for JSON marshalling (it has to do with the way it works)
// Note that we will store both UUID -> AvatarName *and* AvatarName -> UUID on the same database,
//
//	thus the apparent redundancy in fields! (gwyneth 20211030)
//
// The 'validate' decorator is for usage with the go-playground validator, currently unused (gwyneth 20211031)
type avatarUUID struct {
	AvatarName string `json:"name" form:"name" binding:"required" validate:"omitempty,alphanum"`
	UUID       string `json:"key"  form:"key"  binding:"required" validate:"omitempty,uuid4_rfc4122"`
	Grid       string `json:"grid" form:"grid" validate:"omitempty,alphanum"`
}

/*
				  .__
  _____ _____  |__| ____
 /		  \\__	 \ |	 |/	\
|  Y Y	\/ __ \|	 |		 |	 \
|__|_|	(____  /__|___|	 /
	  \/	 \/			  \/
*/

// Configuration options.
type goslConfigOptions struct {
	BATCH_BLOCK                             int		// how many entries to write to the database as a block; the bigger, the faster, but the more memory it consumes.
	loopBatch								int		// how many entries to skip when emitting debug messages in a tight loop.
	noMemory, isServer, isShell             bool	// !isServer && !isShell => FastCGI!
	myDir, myPort, importFilename, database string
	configFilename							string	// name (+ path?) of the configuratio file.
	dbNamePath                              string	// for BuntDB.
	logLevel, logFilename                   string	// for logs.
	maxSize, maxBackups, maxAge             int		// logs configuration options.
}

var goslConfig goslConfigOptions	// list of all configuration opions.
var kv *badger.DB					// current KV store being used (Badger).

// loadConfiguration reads our configuration from a `config.ini` file,
func loadConfiguration() {
	fmt.Println("Reading gosl-basic configuration:") // note that we might not have go-logging active as yet, so we use fmt and write to stdout
	// Open our config file and extract relevant data from there
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("error reading config file %q, falling back to defaults - error was: %s\n", goslConfig.configFilename, err)
		// we fall back to what we have
	}
	// NOTE(gwyneth): the authors of say that 100000 is way too much for Badger.
	// Let's see what happens with BuntDB
	viper.SetDefault("config.BATCH_BLOCK", 100000)
	goslConfig.BATCH_BLOCK = viper.GetInt("config.BATCH_BLOCK")
	viper.SetDefault("config.loopBatch", 1000)
	goslConfig.loopBatch = viper.GetInt("config.loopBatch")
	viper.SetDefault("config.myPort", 3000)
	goslConfig.myPort = viper.GetString("config.myPort")
	viper.SetDefault("config.myDir", "slkvdb")
	goslConfig.myDir = viper.GetString("config.myDir")
	viper.SetDefault("config.isServer", false)
	goslConfig.isServer = viper.GetBool("config.isServer")
	viper.SetDefault("config.isShell", false)
	goslConfig.isShell = viper.GetBool("config.isShell")
	viper.SetDefault("config.database", "badger") // currently, badger, boltdb, leveldb.
	goslConfig.database = viper.GetString("config.database")
	viper.SetDefault("options.importFilename", "") // must be empty by default.
	goslConfig.importFilename = viper.GetString("options.importFilename")
	viper.SetDefault("options.noMemory", false)
	goslConfig.noMemory = viper.GetBool("options.noMemory")
	// Logging options
	viper.SetDefault("log.Filename", "gosl.log")
	goslConfig.logFilename = viper.GetString("log.Filename")
	viper.SetDefault("log.logLevel", "ERROR")
	goslConfig.logLevel = viper.GetString("log.logLevel")
	viper.SetDefault("log.MaxSize", 10)
	goslConfig.maxSize = viper.GetInt("log.MaxSize")
	viper.SetDefault("log.MaxBackups", 3)
	goslConfig.maxBackups = viper.GetInt("log.MaxBackups")
	viper.SetDefault("log.MaxAge", 28)
	goslConfig.maxAge = viper.GetInt("log.MaxAge")
}

// main() starts here.
func main() {
	// Config viper, which reads in the configuration file every time it's needed.
	// Note that we need some hard-coded variables for the path and config file name.
	viper.SetDefault(goslConfig.configFilename, "config.ini")
	viper.SetConfigName(goslConfig.configFilename)
	// just to make sure; it's the same format as OpenSimulator (or MySQL) config files.
	viper.SetConfigType("ini")
	// optionally, look for config in the working directory.
	viper.AddConfigPath(".")
	// this is also a great place to put standard configurations:
	// NOTE:
	viper.AddConfigPath(filepath.Join("$HOME/.config/", os.Args[0]))
	// last chance — check on the usual place for Go source.
	viper.AddConfigPath("$HOME/go/src/git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics/")

	loadConfiguration()

	// Flag setup; can be overridden by config file.
	goslConfig.myPort =			*flag.StringP("port", "p", "3000", "Server port")
	goslConfig.myDir =			*flag.String("dir", "slkvdb", "Directory where database files are stored")
	goslConfig.isServer =		*flag.Bool("server", false, "Run as server on port " + goslConfig.myPort)
	goslConfig.isShell =		*flag.Bool("shell", false, "Run as an interactive shell")
	goslConfig.importFilename = *flag.StringP("import", "i", "", "Import database from W-Hat (use the csv.bz2 versions)")
	goslConfig.configFilename =	*flag.String("config", "config.ini", "Configuration filename [extension defines type, INI by default]")
	goslConfig.database = 		*flag.String("database", "badger", "Database type [badger | buntdb | leveldb]")
	goslConfig.noMemory = 		*flag.Bool("nomemory", true, "Attempt to use only disk to save memory on Badger (important for shared webservers)")
	goslConfig.logLevel =		*flag.StringP("debug", "d", "ERROR", "Logging level, e.g. one of [DEBUG | ERROR | NOTICE | INFO]")
	goslConfig.loopBatch =		*flag.IntP("loopbatch", "l", 1000, "How many entries to skip when emitting debug messages in a tight loop. Only useful when importing huge databases with high logging levels. Set to 1 if you wish to see logs for all entries.")
	goslConfig.BATCH_BLOCK = 	*flag.IntP("batchblock", "b", 100000, "How many entries to write to the database as a block; the bigger, the faster, but the more memory it consumes.")

	// default is FastCGI
	flag.Parse()
	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		fmt.Printf("error parsing/binding flags: %s\n", err)
	}

	if goslConfig.configFilename != "config.ini" {
		viper.SetConfigName(goslConfig.configFilename)
		// we can switch filetypes here
		ext := filepath.Ext(goslConfig.configFilename)[1:]
		viper.SetConfigType(ext)
		// Find and read the config fil
		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("error reading config file %q [type %s], falling back to defaults - error was: %s\n", goslConfig.configFilename, ext, err)
			// we fall back to what we have
		}
	}

	// Avoid division by zero...
	if goslConfig.BATCH_BLOCK < 1 {
		goslConfig.BATCH_BLOCK = 1
	}
	if goslConfig.loopBatch < 1 {
		goslConfig.loopBatch = 1
	}

	// this will allow our configuration file to be 'read on demand'
	// TODO(gwyneth): There is something broken with this, no reason why... (gwyneth 20211026)
	// viper.WatchConfig()
	// viper.OnConfigChange(func(e fsnotify.Event) {
	// 	if goslConfig.isServer || goslConfig.isShell {
	// 		fmt.Println("Config file changed:", e.Name)	// BUG(gwyneth): FastCGI cannot write to output
	// 	}
	// 	loadConfiguration()
	// })

	// NOTE(gwyneth): We cannot write to stdout if we're running as FastCGI, only to logs!
	if goslConfig.isServer || goslConfig.isShell {
		fmt.Println(os.Args[0], " is starting...")
	}

	// This is mostly to deal with scoping issues below. (gwyneth 20211106)
	var err error

	// Setup the lumberjack rotating logger. This is because we need it for the go-logging logger when writing to files. (20170813)
	rotatingLogger := &lumberjack.Logger{
		Filename:   goslConfig.logFilename,
		MaxSize:    goslConfig.maxSize, // megabytes
		MaxBackups: goslConfig.maxBackups,
		MaxAge:     goslConfig.maxAge, //days
	}

	// Set formatting for stderr and file (basically the same).
	logFormat := logging.MustStringFormatter(`%{color}%{time:2006/01/02 15:04:05.0} %{shortfile} - %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`) // must be initialised or all hell breaks loose

	// Setup the go-logging Logger. Do **not** log to stderr if running as FastCGI!
	backendFile := logging.NewLogBackend(rotatingLogger, "", 0)
	backendFileFormatter := logging.NewBackendFormatter(backendFile, logFormat)
	backendFileLeveled := logging.AddModuleLevel(backendFileFormatter)

	theLogLevel, err := logging.LogLevel(goslConfig.logLevel)
	if err != nil {
		log.Warningf("could not set log level to %q — invalid?\nlogging.LogLevel() returned error %q\n", goslConfig.logLevel, err)
	} else {
		log.Debugf("requested file log level: %q\n", theLogLevel.String())
		backendFileLeveled.SetLevel(theLogLevel, "gosl") // we just send debug data to logs if we run asshell
		log.Debugf("file log level set to: %v\n", backendFileLeveled.GetLevel("gosl"))
	}

	if goslConfig.isServer || goslConfig.isShell {
		backendStderr := logging.NewLogBackend(os.Stderr, "", 0)
		backendStderrFormatter := logging.NewBackendFormatter(backendStderr, logFormat)
		backendStderrLeveled := logging.AddModuleLevel(backendStderrFormatter)
		log.Debugf("requested stderr log level: %q\n", theLogLevel.String())
		backendStderrLeveled.SetLevel(theLogLevel, "gosl")
		log.Debugf("stderr log level set to: %v\n", backendStderrLeveled.GetLevel("gosl"))
	}
	/*
			// deprecated, now we set it explicitly if desired
			if goslConfig.isShell {
				backendStderrLeveled.SetLevel(logging.DEBUG, "gosl")	// shell is meant to be for debugging mostly
			} else {
				backendStderrLeveled.SetLevel(logging.INFO, "gosl")
			}
			logging.SetBackend(backendStderrLeveled, backendFileLeveled)
		} else {
			logging.SetBackend(backendFileLeveled)	// FastCGI only logs to file
		}
	*/

	log.Debugf("Full config: %+v\n", goslConfig)

	// Check if this directory actually exists; if not, create it. Panic if something wrong happens (we cannot proceed without a valid directory for the database to be written)
	if stat, err := os.Stat(goslConfig.myDir); err == nil && stat.IsDir() {
		// path is a valid directory
		log.Debugf("valid directory: %q\n", goslConfig.myDir)
	} else {
		// try to create directory
		if err = os.Mkdir(goslConfig.myDir, 0700); err != nil {
			if err != os.ErrExist {
				checkErr(err)
			} else {
				log.Debugf("directory %q exists, no need to create it\n", goslConfig.myDir)
			}
		}
		log.Debugf("created new directory: %q\n", goslConfig.myDir)
	}

	// Special options configuration.
	// Currently, this is only needed for Badger v3, the others have much simpler configurations.
	// (gwyneth 20211106)
	switch goslConfig.database {
	case "badger":
		// Badger v3 - fully rewritten configuration (much simpler!!) (gwyneth 20211026)
		if goslConfig.noMemory {
			// use disk; note that unlike the others, Badger generates its own filenames,
			// we can only pass a _directory_... (gwyneth 20211027)
			goslConfig.dbNamePath = filepath.Join(goslConfig.myDir, databaseName)
			// try to create directory
			if err = os.Mkdir(goslConfig.dbNamePath, 0700); err != nil {
				if err != os.ErrExist {
					checkErr(err)
				} else {
					log.Debugf("directory %q exists, no need to create it\n", goslConfig.dbNamePath)
				}
			} else {
				log.Debugf("created new directory: %q\n", goslConfig.dbNamePath)
			}

			Opt = badger.DefaultOptions(goslConfig.dbNamePath)
			log.Debugf("entering disk mode, Opt is %+v\n", Opt)
		} else {
			// Use only memory
			Opt = badger.LSMOnlyOptions("").WithInMemory(true)
			Opt.WithLevelSizeMultiplier(1)
			Opt.WithNumMemtables(1)
			Opt.WithValueDir(Opt.Dir) // probably not needed
			log.Debugf("entering memory-only mode, Opt is %+v\n", Opt)
		}
		// common config
		Opt.WithLogger(log) // set the internal logger to our own rotating logger
		Opt.WithLoggingLevel(badger.ERROR)
		goslConfig.BATCH_BLOCK = 1000 // try to import less at each time, it will take longer but hopefully work
		log.Info("trying to avoid too much memory consumption")
		// the other databases do not require any special configuration (for now)
	} // /switch

	// if importFilename isn't empty, this means we potentially have something to import.
	if goslConfig.importFilename != "" {
		log.Info("attempting to import", goslConfig.importFilename, "...")
		importDatabase(goslConfig.importFilename)
		log.Info("database finished import.")
	} else {
		// it's not an error if there is no name2key database available for import (gwyneth 20211027)
		log.Debug("no database configured for import — 🆗")
	}

	// Prepare testing data! (common to all database types)
	// Note: this only works for shell/server; for FastCGI it's definitely overkill (gwyneth 20211106),
	//  so we do it only for server/shell mode.
	if goslConfig.isServer || goslConfig.isShell {
		const testAvatarName = "Nobody Here"

		log.Infof("%s started and logging is set up. Proceeding to test database (%s) at %q\n", os.Args[0], goslConfig.database, goslConfig.myDir)
		// generate a random UUID (gwyneth2021103) (gwyneth 20211031)

		var (
			testUUID  = uuid.New().String() // Random UUID (gwyneth 20211031 — from )
			testValue = avatarUUID{testAvatarName, testUUID, "all grids"}
		)
		jsonTestValue, err := json.Marshal(testValue)
		checkErrPanic(err) // something went VERY wrong

		// KVDB Initialisation & Tests
		// Each case is different
		switch goslConfig.database {
		case "badger":
			// Opt has already been set earlier. (gwyneth 20211106)
			kv, err := badger.Open(Opt)
			checkErrPanic(err) // should probably panic, cannot prep new database
			txn := kv.NewTransaction(true)
			err = txn.Set([]byte(testAvatarName), jsonTestValue)
			checkErrPanic(err)
			err = txn.Set([]byte(testUUID), jsonTestValue)
			checkErrPanic(err)
			err = txn.Commit()
			checkErrPanic(err)
			log.Debugf("badger SET %+v (json: %v)\n", testValue, string(jsonTestValue))
			kv.Close()
		case "buntdb":
			goslConfig.dbNamePath = filepath.Join(goslConfig.myDir, databaseName)
			db, err := buntdb.Open(goslConfig.dbNamePath)
			checkErrPanic(err)
			err = db.Update(func(tx *buntdb.Tx) error {
				_, _, err := tx.Set(testAvatarName, string(jsonTestValue), nil)
				return err
			})
			checkErr(err)
			log.Debugf("buntdb SET %+v (json: %v)\n", testValue, string(jsonTestValue))
			db.Close()
		case "leveldb":
			goslConfig.dbNamePath = filepath.Join(goslConfig.myDir, databaseName)
			db, err := leveldb.OpenFile(goslConfig.dbNamePath, nil)
			checkErrPanic(err)
			err = db.Put([]byte(testAvatarName), jsonTestValue, nil)
			checkErrPanic(err)
			log.Debugf("leveldb SET %+v (json: %v)\n", testValue, string(jsonTestValue))
			db.Close()
		} // /switch
		// common to all databases:
		key, grid := searchKVname(testAvatarName)
		log.Debugf("GET %q returned %q [grid %q]\n", testAvatarName, key, grid)
		log.Info("KV database seems fine.")

		if goslConfig.importFilename != "" {
			log.Info("attempting to import", goslConfig.importFilename, "...")
			importDatabase(goslConfig.importFilename)
			log.Info("database finished import.")
		} else {
			// it's not an error if there is no name2key database available for import (gwyneth 20211027)
			log.Debug("no database configured for import")
		}
	}

	if goslConfig.isShell {
		log.Info("starting to run as interactive shell")
		fmt.Println("Ctrl-C to quit, or just type \"quit\".")
		var err error // to avoid assigning text in a different scope (this is a bit awkward, but that's the problem with bi-assignment)
		var avatarName, avatarKey, gridName string

		rl, err := readline.New("enter avatar name or UUID: ")
		if err != nil {
			log.Criticalf("major readline issue preventing normal functioning; error was %q\n", err)
		}
		defer rl.Close()

		for {
			checkInput, err := rl.Readline()
			if err != nil || checkInput == "quit" { // io.EOF
				break
			}
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
				fmt.Println("sorry, unknown input", checkInput)
			}
		}
		// never leaves until Ctrl-C or by typing `quit`. (gwyneth 20211106)
		log.Debug("interactive session finished.")
	} else if goslConfig.isServer {
		// set up routing.
		// NOTE(gwyneth): one function only because FastCGI seems to have problems with multiple handlers.
		http.HandleFunc("/", handler)
		log.Debug("directory for database:", goslConfig.myDir)

		log.Info("starting to run as web server on port :" + goslConfig.myPort)
		err := http.ListenAndServe(":"+goslConfig.myPort, nil) // set listen port
		checkErrPanic(err)                                     // if it can't listen to all the above, then it has to abort anyway
	} else {
		// default is to run as FastCGI!
		// works like a charm thanks to http://www.dav-muz.net/blog/2013/09/how-to-use-go-and-fastcgi/
		log.Debug("http.DefaultServeMux is", http.DefaultServeMux)
		log.Info("Starting to run as FastCGI")
		if err := fcgi.Serve(nil, http.HandlerFunc(handler)); err != nil {
			log.Errorf("seems that we got an error from FCGI: %q\n", err)
			checkErrPanic(err)
		}
	}

	// we should never have reached this point!
	log.Error("unknown usage — this application may run as a standalone server, as a FastCGI application, or as an interactive shell")
	if goslConfig.isServer || goslConfig.isShell {
		flag.PrintDefaults()
	}
}

// handler deals with incoming queries and/or associates avatar names with keys depending on parameters.
// Basically we check if both an avatar name and a UUID key has been received: if yes, this means a new entry;
// -	if just the avatar name was received, it means looking up its key;
// -	if just the key was received, it means looking up the name (not necessary since llKey2Name does that, but it's just to illustrate);
//   - if nothing is received, then return an error
//
// Note: to ensure quick lookups, we actually set *two* key/value pairs, one with avatar name/UUID,
// the other with UUID/name — that way, we can efficiently search for *both* in the same database!
// Theoretically, we could even have *two* KV databases, but that's too much trouble for the
// sake of some extra efficiency (gwyneth 20211030)
func handler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		logErrHTTP(w, http.StatusNotFound, "no avatar and/or UUID received")
		return
	}
	// test first if this comes from Second Life or OpenSimulator
	/*
		if r.Header.Get("X-Secondlife-Region") == "" {
			logErrHTTP(w, http.StatusForbidden, "Sorry, this application only works inside Second Life.")
			return
		}
	*/
	name	:= r.Form.Get("name")	// can be empty.
	key		:= r.Form.Get("key")	// can be empty.
	compat	:= r.Form.Get("compat")	// compatibility mode with W-Hat,
	var uuidToInsert avatarUUID
	messageToSL := "" // this is what we send back to SL - defined here due to scope issues.
	if name != "" {
		if key != "" {
			// we received both: add a new entry.
			uuidToInsert.UUID = key
			uuidToInsert.Grid = r.Header.Get("X-Secondlife-Shard")
			jsonUUIDToInsert, err := json.Marshal(uuidToInsert)
			checkErr(err)
			switch goslConfig.database {
			case "badger":
				kv, err := badger.Open(Opt)
				checkErrPanic(err) // should probably panic.
				txn := kv.NewTransaction(true)
				defer txn.Discard()
				err = txn.Set([]byte(name), jsonUUIDToInsert)
				checkErrPanic(err)
				err = txn.Commit()
				checkErrPanic(err)
				kv.Close()
			case "buntdb":
				db, err := buntdb.Open(goslConfig.dbNamePath)
				checkErrPanic(err)
				defer db.Close()
				err = db.Update(func(tx *buntdb.Tx) error {
					_, _, err := tx.Set(name, string(jsonUUIDToInsert), nil)
					return err
				})
				checkErr(err)
			case "leveldb":
				db, err := leveldb.OpenFile(goslConfig.dbNamePath, nil)
				checkErrPanic(err)
				err = db.Put([]byte(name), jsonUUIDToInsert, nil)
				checkErrPanic(err)
				db.Close()
			}
			messageToSL += "Added new entry for '" + name + "' which is: " + uuidToInsert.UUID + " from grid: '" + uuidToInsert.Grid + "'"
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
		// in this scenario, we have the UUID key but no avatar name: do the equivalent of a llKey2Name
		name, grid := searchKVUUID(key)
		if compat == "false" {
			messageToSL += "avatar name for " + key + "' is '" + name + "' on grid: '" + grid + "'"
		} else { // empty also means true!
			messageToSL += name
		}
	} else {
		// neither UUID key nor avatar received, this is an error
		logErrHTTP(w, http.StatusNotFound, "empty avatar name and UUID key received, cannot proceed")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, messageToSL)
}
