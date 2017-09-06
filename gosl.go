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
	"github.com/dgraph-io/badger/table"
	"github.com/op/go-logging"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
//	"io/ioutil"
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

// Logging setup.
var log = logging.MustGetLogger("gosl")	// configuration for the go-logging logger, must be available everywhere
var logFormat logging.Formatter

// Opt is used for KV database setup.
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
 
// main() starts here.
func main() {
	// Flag setup
	var myPort	 = flag.String("port", "3000", "Server port")
	var myDir	 = flag.String("dir", "slkvdb", "Directory where database files are stored")
	var isServer = flag.Bool("server", false, "Run as server on port " + *myPort)
	var isShell  = flag.Bool("shell", false, "Run as an interactive shell")
	var importFilename = flag.String("import", "", "Import database from W-Hat (use the csv.bz2 version)")
	var noMemory = flag.Bool("nomemory", false, "Attempt to use only disk to save memory (important for shared webservers)")
	
	// default is FastCGI

	flag.Parse()
	// We cannot write to stdout if we're running as FastCGI, only to logs!
	
	if *isServer || *isShell {
		fmt.Println("gosl is starting...")	
	} else { // FastCGI: we cannot write to stdio, we need to setup the logger so that we can write to disk
		*noMemory = true
	}
	
	// Setup the lumberjack rotating logger. This is because we need it for the go-logging logger when writing to files. (20170813)
	rotatingLogger := &lumberjack.Logger{
		Filename:	 "gosl.log",
		MaxSize:	 10, // megabytes
		MaxBackups: 3,
		MaxAge:	 28, //days
	}
	
	// Set formatting for stderr and file (basically the same).
	logFormat := logging.MustStringFormatter(`%{color}%{time:2006/01/02 15:04:05.0} %{shortfile} - %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`) 	// must be initialised or all hell breaks loose
	
	// Setup the go-logging Logger. Do **not** log to stderr if running as FastCGI!
	backendFile				:= logging.NewLogBackend(rotatingLogger, "", 0)
	backendFileFormatter	:= logging.NewBackendFormatter(backendFile, logFormat)
	backendFileLeveled 		:= logging.AddModuleLevel(backendFileFormatter)
	backendFileLeveled.SetLevel(logging.INFO, "gosl")	// we just send debug data to logs if we run as shell
	
	if *isServer || *isShell {
		backendStderr			:= logging.NewLogBackend(os.Stderr, "", 0)
		backendStderrFormatter	:= logging.NewBackendFormatter(backendStderr, logFormat)
		backendStderrLeveled 	:= logging.AddModuleLevel(backendStderrFormatter)
		if *isShell {
			backendStderrLeveled.SetLevel(logging.DEBUG, "gosl")	// shell is meant to be for debugging mostly
		} else {
			backendStderrLeveled.SetLevel(logging.INFO, "gosl")
		}
		logging.SetBackend(backendStderrLeveled, backendFileLeveled)
	} else {
		logging.SetBackend(backendFileLeveled)	// FastCGI only logs to file
	}

	log.Info("gosl started and logging is set up. Proceeding to test KV database.")
	const testAvatarName = "Nobody Here"
	var err error
	Opt = badger.DefaultOptions
	// Check if this directory actually exists; if not, create it. Panic if something wrong happens (we cannot proceed without a valid directory for the database to be written
	if stat, err := os.Stat(*myDir); err == nil && stat.IsDir() {
		// path is a valid directory
		log.Debugf("Valid directory: %s\n", *myDir)
	} else {
		// try to create directory
		err = os.Mkdir(*myDir, 0700)
		checkErrPanic(err) // cannot make directory, panic and exit logging what went wrong
		log.Debugf("Created new directory: %s\n", *myDir)		
	}
	Opt.Dir = *myDir
	Opt.ValueDir = Opt.Dir
	if *noMemory {
		Opt.MapTablesTo = table.Nothing
		log.Info("Trying to avoid too much memory consumption")	
	}
	kv, err := badger.NewKV(&Opt)
	checkErrPanic(err) // should probably panic, cannot prep new database
	var testValue = avatarUUID{ NullUUID, "all grids" }
	jsonTestValue, err := json.Marshal(testValue)
	checkErrPanic(err) // something went VERY wrong
	kv.Set([]byte(testAvatarName), jsonTestValue, 0x00)
	log.Debugf("SET %+v (json: %v)\n", testValue, string(jsonTestValue))
	kv.Close()
	key, grid := searchKVname(testAvatarName)
	log.Debugf("GET '%s' returned '%s' [grid '%s']\n", testAvatarName, key, grid)
	
	log.Info("KV database seems fine.")
	
	if *importFilename != "" {
		log.Info("Attempting to import", *importFilename, "...")
		importDatabase(*importFilename)
		log.Info("Database finished import.")
	}
	
	if (*isShell) {
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
	
	if (*isServer) {
		log.Info("Starting to run as web server on port " + *myPort)
		err := http.ListenAndServe(":" + *myPort, nil) // set listen port
		checkErrPanic(err) // if it can't listen to all the above, then it has to abort anyway
	} else {
		// default is to run as FastCGI!
		// works like a charm thanks to http://www.dav-muz.net/blog/2013/09/how-to-use-go-and-fastcgi/
		log.Info("Starting to run as FastCGI")
		log.Info("http.DefaultServeMux is", http.DefaultServeMux)
		if err := fcgi.Serve(nil, nil); err != nil {
			checkErrPanic(err)
		}
	}
	// we should never have reached this point!
	log.Error("Unknown usage! This application may run as a standalone server, as FastCGI application, or as an interactive shell")
	if *isServer || *isShell {
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
	if r.Header.Get("X-Secondlife-Region") == "" {
		logErrHTTP(w, http.StatusForbidden, "Sorry, this application only works inside Second Life.")
		return
	}
	name := r.Form.Get("name") // can be empty
	key := r.Form.Get("key") // can be empty
	compat := r.Form.Get("compat") // compatibility mode with W-Hat
	var valueToInsert avatarUUID
	messageToSL := "" // this is what we send back to SL - defined here due to scope issues.
	if name != "" {
		if key != "" {
			// we received both: add a new entry
			kv, err := badger.NewKV(&Opt)
			checkErrPanic(err) // should probably panic
			valueToInsert.UUID = key
			valueToInsert.Grid = r.Header.Get("X-Secondlife-Shard")
			jsonValueToInsert, err := json.Marshal(valueToInsert)
			checkErr(err)
			kv.Set([]byte(name), jsonValueToInsert, 0x00)
			kv.Close()
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
	kv, err := badger.NewKV(&Opt)
	defer kv.Close()
	var item badger.KVItem
	if err := kv.Get([]byte(avatarName), &item); err != nil {
		log.Errorf("Error while getting name: %s - %v\n", avatarName, err)
		return NullUUID, ""
	}
	var val avatarUUID
	if err = json.Unmarshal(item.Value(), &val); err != nil {
		log.Errorf("Error while unparsing UUID for name: %s - %v\n", avatarName, err)
		return NullUUID, ""
	}
	return val.UUID, val.Grid
}
// searchKVUUID searches the KV database for an avatar key.
func searchKVUUID(avatarKey string) (name string, grid string) {
	kv, err := badger.NewKV(&Opt)
	checkErr(err) // should probably panic

	itOpt := badger.DefaultIteratorOptions
	itr := kv.NewIterator(itOpt)
	var val = avatarUUID{ NullUUID, "" }
	var found string
	checks := 0
	time_start := time.Now()
	for itr.Rewind(); itr.Valid(); itr.Next() {
		item := itr.Item()
		if err = json.Unmarshal(item.Value(), &val); err == nil {
			checks++	//Just to see how many
			if avatarKey == val.UUID {	// are these pointers?
				found = string(item.Key())
				break
			}
		}	
	}
	time_end := time.Now()
	diffTime := time_end.Sub(time_start)
	log.Debugf("Made %d checks for '%s' in %v\n", checks, avatarKey, diffTime)
	itr.Close()
	kv.Close()
	return found, val.Grid
}

// importDatabase is essentially reading a bzip2'ed CSV file with UUID,AvatarName downloaded from http://w-hat.com/#name2key .
//	One could theoretically set a cron job to get this file, save it on disk periodically, and keep the database up-to-date
//	see https://stackoverflow.com/questions/24673335/how-do-i-read-a-gzipped-csv-file for the actual usage of these complicated things!
func importDatabase(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	gr := bzip2.NewReader(f) // open bzip2 reader
	cr := csv.NewReader(gr)  // open csv reader and feed the bzip2 reader into it
	limit := 0
	kv, err := badger.NewKV(&Opt)
	checkErrPanic(err) // should probably panic		
	defer kv.Close()
	time_start := time.Now()
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println("Key:", record[0], "Name:", record[1])			
		jsonNewEntry, err := json.Marshal(avatarUUID{ record[0], "Production" }) // W-Hat keys come all from the main LL grid, known as 'Production'
		if err != nil {
			log.Warning(err)
		} else {
			kv.Set([]byte(record[1]), []byte(jsonNewEntry), 0x00)
		}
		limit++
		if limit % 1000000 == 0 {
			time_end := time.Now()
			diffTime := time_end.Sub(time_start)
			log.Info("Read", limit, "records (or thereabouts) in", diffTime)
		}
	}
	time_end := time.Now()
	diffTime := time_end.Sub(time_start)
	log.Info("Total read", limit, "records (or thereabouts) in", diffTime)
}

// NOTE(gwyneth):Auxiliary functions which I'm always using...

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