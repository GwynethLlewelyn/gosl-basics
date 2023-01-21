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
		pc, file, line, ok := runtime.Caller(1)
		log.Panicf("%s:%d (%v) [ok: %v] - panic: %v\n", filepath.Base(file), line, pc, ok, err)
	}
}

// checkErr checks if there is an error, and if yes, it logs it out and continues.
//
//	this is for 'normal' situations when we want to get a log if something goes wrong but do not need to panic
func checkErr(err error) {
	if err != nil {
		pc, file, line, ok := runtime.Caller(1)
		// log.Error(filepath.Base(file), ":", line, ":", pc, ok, " - error:", err)
		log.Errorf("%s:%d (%v) [ok: %v] - error: %v\n", filepath.Base(file), line, pc, ok, err)
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
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}

// isValidUUID checks if the UUID is valid.
// Thanks to Patrick D'Appollonio https://stackoverflow.com/questions/25051675/how-to-validate-uuid-v4-in-go
//  as well as https://stackoverflow.com/a/46315070/1035977 (gwyneth 29211031)
/*
// Deprecated, since regexps may be overkill here; Google's own package is far more efficient and we'll use it directly (gwyneth 20211031)
func isValidUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}
*/
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
