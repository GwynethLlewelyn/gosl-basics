// Auxiliary functions which I'm always using.
// Moved to separate file on 20211102.
package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	// "github.com/op/go-logging"
)

// checkErrPanic logs a fatal error and panics.
func checkErrPanic(err error) {
	if err != nil {
		if pc, file, line, ok := runtime.Caller(1); ok {
			log.Panicf("%s:%d (%v) - panic: %v\n", filepath.Base(file), line, pc, err)
			return
		}
		log.Panic(err)
	}
}

// checkErr checks if there is an error, and if yes, it logs it out and continues.
//
//	this is for 'normal' situations when we want to get a log if something goes wrong but do not need to panic
func checkErr(err error) {
	if err != nil {
		if pc, file, line, ok := runtime.Caller(1); ok {
			log.Errorf("%s:%d (%v) - error: %v\n", filepath.Base(file), line, pc, err)
			return
		}
		log.Panic(err)
	}
}

// Auxiliary functions for HTTP handling

// checkErrHTTP returns an error via HTTP and also logs the error.
func checkErrHTTP(w http.ResponseWriter, httpStatus int, errorMessage string, err error) {
	if err != nil {
		http.Error(w, fmt.Sprintf(errorMessage, err), httpStatus)
		if pc, file, line, ok := runtime.Caller(1); ok {
			log.Error("(", http.StatusText(httpStatus), ") ", filepath.Base(file), ":", line, ":", pc, " - error:", errorMessage, err)
			return
		}
		log.Error("(", http.StatusText(httpStatus), ") ", errorMessage, err)
	}
}

// checkErrPanicHTTP returns an error via HTTP and logs the error with a panic.
func checkErrPanicHTTP(w http.ResponseWriter, httpStatus int, errorMessage string, err error) {
	if err != nil {
		http.Error(w, fmt.Sprintf(errorMessage, err), httpStatus)
		if pc, file, line, ok := runtime.Caller(1); ok {
			log.Panic("(", http.StatusText(httpStatus), ") ", filepath.Base(file), ":", line, ":", pc, " - panic:", errorMessage, err)
			return
		}
		log.Panic("(", http.StatusText(httpStatus), ") ", errorMessage, err)
	}
}

// logErrHTTP assumes that the error message was already composed and writes it to HTTP and logs it.
//
//	this is mostly to avoid code duplication and make sure that all entries are written similarly
func logErrHTTP(w http.ResponseWriter, httpStatus int, errorMessage string) {
	http.Error(w, errorMessage, httpStatus)
	log.Error("(" + http.StatusText(httpStatus) + ") " + errorMessage)
}

// funcName is @Sonia's solution to get the name of the function that Go is currently running.
//
//	This will be extensively used to deal with figuring out where in the code the errors are!
//	Source: https://stackoverflow.com/a/10743805/1035977 (20170708)
func funcName() string {
	if pc, _, _, ok := runtime.Caller(1); ok {
		return runtime.FuncForPC(pc).Name()
	}
	return ""
}

// isValidUUID returns whether the UUID is valid.
// Thanks to Patrick D'Appollonio https://stackoverflow.com/questions/25051675/how-to-validate-uuid-v4-in-go
//  as well as https://stackoverflow.com/a/46315070/1035977 (gwyneth 29211031)
// This exists mostly to be able to return just _one_ value (the boolean) and not require anything else.
// Also note that _some_ UUIDs are not fully v4 compliant; LL invented a few ones for the first "special" residents
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
