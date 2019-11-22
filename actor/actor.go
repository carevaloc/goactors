package actor

import (
	"io"
	"io/ioutil"
	"log"
)

// DefaultInCap is the default capacity of the In channel
const DefaultInCap = 100

// Actor is the base type of all actors
type Actor struct {
	In     chan interface{}
	StopCh chan struct{}
}

// InCapacity returns the capacity that the In channel wil have
func (ba Actor) InCapacity() int {
	return DefaultInCap
}

// Log is the Logger used to write output messages
var Log = log.New(ioutil.Discard, "goact: ", log.Ldate|log.Ltime)

// SetLogOutput sets the output destination for the logger.
func SetLogOutput(w io.Writer) {
	Log.SetOutput(w)
}
