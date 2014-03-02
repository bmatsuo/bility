// csvstream.go [created: Sat,  1 Mar 2014]

/*
Package csvstream provides simple asynchronous streaming for CSV data.
It requires continuation logic to be implemented by the application as
errors depend of the configuration of csv.Reader structs.
*/
package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
)

type StreamRow struct {
	Cols []string
	Err  error
}

type Stream <-chan *StreamRow

// read from r asynchronously. the returned channel is closed on EOF.
// if there is an error reading from r, the error is passed over the
// channel and the channel is closed. It is up to the application to
// decide if a new Stream should be created to continue reading from
// r.
func NewStream(r *csv.Reader, bufsize uint) Stream {
	ch := make(chan *StreamRow, bufsize)
	go func() {
		defer close(ch)
		for {
			cols, err := r.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				ch <- &StreamRow{nil, err}
				return
			}
			ch <- &StreamRow{cols, nil}
		}
		panic("unreachable")
	}()
	return ch
}

var UnknownColumnErr = fmt.Errorf("unknown column")
var MissingColumnErr = fmt.Errorf("missing column")

// Returns UnknownColumnErr if name does not exist in header.
// Returns MissingColumnErr if cols does not contain a name column (not enough
// columns).
func GetColumn(header Header, cols []string, name string) (string, error) {
	i, ok := header[name]
	if !ok {
		return "", UnknownColumnErr
	}
	if i >= len(cols) {
		return "", MissingColumnErr
	}
	return cols[i], nil
}

type Header map[string]int

func ReadHeader(r *csv.Reader) (Header, error) {
	cols, err := r.Read()
	if err != nil { // EOF too
		return nil, err
	}
	header := make(map[string]int, len(cols))
	for i := range cols {
		// assumes no collisions
		header[cols[i]] = i
	}
	return header, nil
}
