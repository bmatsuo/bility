/*
I'm actually not sure yet that I've computed the tag changes correctly.
I'm only using EC2 instance usage items and using only using their usage
start time.  Further, the resource id is used to collate rows. It may be
that these ids can be found elsewhere in the document and used to collect
more information about instances. But is it redundant?
*/
package main

import (
	"github.com/bmatsuo/bility/go-awsbilling"
	"github.com/bmatsuo/bility/go-csvutil"

	"encoding/csv"
	"log"
	"os"
	"sync"
	"time"
)

func main() {
	var filename string
	var file *os.File
	switch len(os.Args[1:]) {
	case 1:
		filename = os.Args[1]
	case 0:
		log.Fatal("missing argument")
	default:
		log.Fatal("too many arguments given")
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	header, stream, err := awsbilling.NewCSVStream(file, 1)
	if err != nil {
		log.Fatalf("failed opening data stream; %v", err)
	}

	tags := awsbilling.GetTags(header)

	instanceTags := make(map[string]map[string]TagChain, 0)

	for row := range stream {
		if row.Err != nil {
			log.Fatalf("failed reading csv data; %v", row.Err)
		}

		isInstance, _ := awsbilling.IsEC2InstanceBillingItem(header, row.Cols)
		if !isInstance {
			continue
		}
		_start, err := csvutil.GetColumn(header, row.Cols, awsbilling.H_UsageStartDate)
		if err != nil {
			log.Print("unable to locate usage start date: ", err)
			continue
		}
		start, err := awsbilling.ParseTime(_start)
		if err != nil {
			log.Println("unable to parse start date %q: ", _start, err)
		}

		instance, err := csvutil.GetColumn(header, row.Cols, awsbilling.H_ResourceId)
		if err != nil {
			log.Println("unable to locate resource id: %v", row.Cols)
		}

		for _, tag := range tags {
			name := tag.Name()
			// XXX treats missing values as empty strings
			val, _ := csvutil.GetColumn(header, row.Cols, tag.CSVHeader())
			tags := instanceTags[instance]
			if tags == nil {
				tags = make(map[string]TagChain)
				instanceTags[instance] = tags
			}
			chain := tags[name]
			if chain == nil {
				chain = newTagChain()
				tags[name] = chain
			}
			chain.Record(start, val)
		}

	}

	w := csv.NewWriter(os.Stdout)
	defer func() {
		w.Flush()
		err := w.Error()
		if err != nil  {
			log.Print("error writing csv: ", err)
		}
	}()
	w.Write([]string{"Instance", "Tag", "Time", "Value", "PreviousValue"})
	for instance, tags := range instanceTags {
		for tag, chain := range tags {
			var last TagRecord
			for i, rec := range chain.Transitions() {
				if i > 0 {
					w.Write([]string{
						instance,
						tag,
						rec.Time.Format(time.RFC3339),
						rec.TagValue,
						last.TagValue,
					})
				}
				last = rec
			}
		}
	}
}

type TagChain interface {
	Record(t time.Time, value string)
	Transitions() []TagRecord
}

// a threadsafe chain of tags transitions.
type tagChain struct {
	lock *sync.Mutex
	recs []*TagRecord
}

type TagRecord struct {
	Time     time.Time
	TagValue string
}

func newTagChain() *tagChain {
	return &tagChain{
		lock: new(sync.Mutex),
	}
}

func (chain *tagChain) Transitions() []TagRecord {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	recs := make([]TagRecord, len(chain.recs))
	for i := range recs {
		recs[i] = *chain.recs[i]
	}

	return recs
}

func (chain *tagChain) Record(t time.Time, value string) {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	if len(chain.recs) == 0 {
		chain.recs = append(chain.recs, &TagRecord{t, value})
		return
	}

	// reverse iteration appears better in terms of average number of iterations
	n := len(chain.recs)
	i := n - 1
	for ; i >= 0; i-- {
		rec := chain.recs[i]
		if rec.Time.Before(t) {
			break
		}
	}

	if i >= 0 && chain.recs[i].TagValue == value {
		// no value transition
		return
	}

	// append/replace/insert
	if i == n-1 {
		chain.recs = append(chain.recs, &TagRecord{t, value})
	} else if chain.recs[i+1].TagValue == value {
		chain.recs[i+1].Time = t
	} else {
		recs := append(chain.recs, nil)
		copy(recs[i+2:], recs[i+1:])
		recs[i+1] = &TagRecord{t, value}
		chain.recs = recs
	}
}

