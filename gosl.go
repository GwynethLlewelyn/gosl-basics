// gosl is a basic example of how to develop external web services for Second Life/OpenSimulator using the Go programming language.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/op/go-logging"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
	"net/http/fcgi"
	"os"
	"path/filepath"
//	"regexp"
	"runtime"
//	"strings"
)

const NullUUID = "00000000-0000-0000-0000-000000000000" // always useful when we deal with SL/OpenSimulator...

// Logging setup	
var log = logging.MustGetLogger("gosl")	// configuration for the go-logging logger, must be available everywhere
var logFormat logging.Formatter

/*
			   .__		  
  _____ _____  |__| ____  
 /	   \\__	 \ |  |/	\ 
|  Y Y	\/ __ \|  |	  |	 \
|__|_|	(____  /__|___|	 /
	  \/	 \/		   \/ 
*/
 
// main() starts here.
func main() {
	// Flag setup
	var myPort	 = flag.String("port", "3000", "Server port")
	var isServer = flag.Bool("server", false, "Run as server on port " + *myPort)
	var isShell  = flag.Bool("shell", false, "Run as an interactive shell")
	// default is FastCGI

	flag.Parse()
	// We cannot write to stdout if we're running as FastCGI, only to logs!
	
	if *isServer || *isShell {
		fmt.Println("gosl is starting...")
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

	log.Info("gosl started and logging is set up. Proceeding to test KVdatabase.")
	
	opt := badger.DefaultOptions
//	dir, _ := ioutil.TempDir("", "gosl.kv")
//	opt.Dir = dir
//	opt.ValueDir = dir
	kv, err := badger.NewKV(&opt)
	checkErr(err) // should probably panic

	key := []byte(NullUUID)

	kv.Set(key, []byte("Nobody Here"), 0x00)
	fmt.Printf("SET %s \n", key)
	defer kv.Close()
	
	log.Info("KV database seems fine.")
	
	if (*isShell) {
		log.Info("Starting to run as interactive shell")
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Ctrl-C to quit.")
		var err error	// to avoid assigning text in a different scope (this is a bit awkward, but that's the problem with bi-assignment)
		var text string
		for {
			// Prompt and read			
			fmt.Print("Enter avatar name: ")
			text, err = reader.ReadString('\n')
			checkErr(err)
			fmt.Println("You typed:", text, "which has", len(text), "character(s).")
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
//  if just the avatar name was received, it means looking up its key;
//  if just the key was received, it means looking up the name (not necessary since llKey2Name does that, but it's just to illustrate);
//  if nothing is received, then return an error
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
	messageToSL := "" // this is what we send back to SL - defined here due to scope issues.
	if name != "" {
		if key != "" {
			// we received both: add a new entry
			messageToSL += "Added new entry for '" + name + "' which is: " + key
			
		} else {
			// we just received the name: look up its UUID key.
			messageToSL += "UUID for '" + name + "' is: " + key
		}
	} else if key != "" {
		// in this scenario, we have the UUID key but no avatar name: do the equivalent of a llKey2Name 
			messageToSL += "Avatar name for " + key + "' is '" + name + "'"

	} else {
		// neither UUID key nor avatar received, this is an error
		logErrHTTP(w, http.StatusNotFound, "Empty avatar name and UUID key received, cannot proceed")
		return	
	}	
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, messageToSL)
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

