package log

import (
	"log"
	"os"
)

var (
	// Info logs a message at level Info.
	Info = log.New(os.Stdout, "🔵 ", log.Ldate|log.Ltime|log.Lshortfile)
	// Warning logs a message at level Warning.
	Warning = log.New(os.Stdout, "🟡 ", log.Ldate|log.Ltime|log.Lshortfile)
	// Error logs a message at level Error.
	Error = log.New(os.Stderr, "🔴 ", log.Ldate|log.Ltime|log.Lshortfile)
)
