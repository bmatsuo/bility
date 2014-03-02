// csvstream.go [created: Sat,  1 Mar 2014]

/*
Package csvstream provides simple asynchronous streaming for CSV data.
It requires continuation logic to be implemented by the application as
errors depend of the configuration of csv.Reader structs.
*/
package csvutil

import (
	"encoding/csv"
	"io"
)

type Row struct {
	Cols []string
	Err  error
}

type Stream <-chan *Row

// read from r asynchronously. the returned channel is closed on EOF.
// if there is an error reading from r, the error is passed over the
// channel and the channel is closed. It is up to the application to
// decide if a new Stream should be created to continue reading from
// r.
func NewStream(r *csv.Reader, bufsize uint) Stream {
	ch := make(chan *Row, bufsize)
	go func() {
		defer close(ch)
		for {
			cols, err := r.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				ch <- &Row{nil, err}
				return
			}
			ch <- &Row{cols, nil}
		}
		panic("unreachable")
	}()
	return ch
}
