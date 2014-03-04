// awsbilling.go [created: Sat,  1 Mar 2014]

/*
Various helper functions for dealing with AWS billing CSV reports.
*/
package awsbilling

import (
	"github.com/bmatsuo/bility/go-csvutil"

	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

func NewCSVReader(r io.Reader) *csv.Reader {
	csvr := csv.NewReader(r)
	csvr.FieldsPerRecord = -1 // some lines have fewer fields than file headers
	return csvr
}

// csv data is streamed from r. no more than bufsize rows will be held w/o being
// processed.
// BUG you cannot stop the stream (except by closing the underlying reader)
func NewCSVStream(r io.Reader, bufsize uint) (csvutil.Header, csvutil.Stream, error) {
	csvr := NewCSVReader(r)
	header, err := csvutil.ReadHeader(csvr)
	if err != nil {
		return nil, nil, err
	}
	stream := csvutil.NewStream(csvr, bufsize)
	return header, stream, nil
}

const (
	H_ProductName    = "ProductName"
	H_Operation      = "Operation"
	H_UsageStartDate = "UsageStartDate"
	H_UsageEndDate   = "UsageEndDate"
	H_UnBlendedCost  = "UnBlendedCost"
	H_ResourceId     = "ResourceId"
)

// determine if the row contains EC2 instance billing.
func IsEC2InstanceBillingItem(header csvutil.Header, cols []string) (bool, error) {
	for _, filter := range []struct{ col, value string }{
		{H_ProductName, "Amazon Elastic Compute Cloud"},
		{H_Operation, "RunInstances"},
	} {
		val, err := csvutil.GetColumn(header, cols, filter.col)
		if err != nil {
			return false, err
		}
		if val != filter.value {
			return false, nil
		}
	}
	return true, nil
}

const TimeFormat = "2006-01-02 15:04:05"

func ParseTime(str string) (time.Time, error) {
	return time.Parse(TimeFormat, str)
}

type Tag struct {
	name      string
	csvheader string
}

const TagPrefix = "user:"

func (t *Tag) Name() string {
	return t.name
}

func (t *Tag) CSVHeader() string {
	return TagPrefix + t.name
}

// extract known tags from a header. there are no guarantees regarding tag order.
func GetTags(header csvutil.Header) []*Tag {
	tags := make([]*Tag, 0, len(header)/2) // arbitrary cap
	for colname := range header {
		tag, err := ParseTag(colname)
		if err == nil {
			tags = append(tags, tag)
		}
	}
	return tags
}

func ParseTag(csvheader string) (*Tag, error) {
	if strings.HasPrefix(csvheader, TagPrefix) {
		tag := &Tag{
			name:      csvheader[len(TagPrefix):],
			csvheader: csvheader,
		}
		return tag, nil
	}
	return nil, fmt.Errorf("not a tag")
}
