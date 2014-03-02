package main

import (
	"encoding/csv"
	"fmt"
	"github.com/bmatsuo/bility/go-csvutil"
	"log"
	"os"
	"io"
	"strings"
	"strconv"
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

	r := csv.NewReader(file)
	r.FieldsPerRecord = -1 // some files have fewer fields than headers

	header, err := csvutil.ReadHeader(r)
	if err != nil {
		log.Fatalf("failed reading csv header; %v", err)
	}

	tags := GetAWSTags(header)

	costs := make(DateTagValueCostTable)

	// process rows as a stream
	for row := range csvutil.NewStream(r, 1) {
		if row.Err != nil {
			log.Fatalf("failed reading csv data; %v", row.Err)
		}

		dailyCost, err := AWSDailyCost(header, row.Cols)
		if err != nil {
			log.Printf("error reading cost: %v (%v)", err, row.Cols)
		}

		for _, tag := range tags {
			val, err := csvutil.GetColumn(header, row.Cols, tag.csvheader)
			if err != nil {
				log.Printf("error reading tag: %v (%v)", err, row.Cols)
			}

			for date, cost := range dailyCost {
				key := [3]string{date, tag.name, val}
				costs[key] += cost
			}
		}
	}

	err = costs.WriteCSV(os.Stdout)
	if err != nil {
		log.Print("output error: ", err)
	}
}

const AWSTimeFormat = "2006-01-02 15:04:05"

func AWSDailyCost(header csvutil.Header, cols []string) (map[string]float64, error) {
	_cost, err := csvutil.GetColumn(header, cols, "UnBlendedCost")
	if err != nil || _cost == "" {
		return nil, nil
	}
	cost, err := strconv.ParseFloat(_cost, 64)
	if err != nil {
		return nil, fmt.Errorf("coludn't parse cost %q: %v", _cost, err)
	}

	_start, err := csvutil.GetColumn(header, cols, "UsageStartDate")
	if err != nil || _start == "" {
		return nil, nil
	}
	start, err := time.Parse(AWSTimeFormat, _start)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse start time %q: %v", _start, err)
	}

	_end, err := csvutil.GetColumn(header, cols, "UsageEndDate")
	if err != nil || _end == "" {
		return nil, nil
	}
	end, err := time.Parse(AWSTimeFormat, _end)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse end time %q: %v", _end, err)
	}

	span := end.Sub(start)
	if span < 0 {
		return nil, fmt.Errorf("start time after end time")
	}

	type date struct {
		year, month, day int
	}
	dateCost := make(map[string]float64)
	today := start
	for done := false; !done; {
		todayEnd := today.Round(24 * time.Hour)
		if todayEnd.Before(today) || todayEnd.Equal(today) {
			todayEnd = todayEnd.Add(24 *time.Hour)
		}
		if end.Before(todayEnd) {
			todayEnd = end
			done = true
		}
		todaySpan := todayEnd.Sub(today)
		todayFrac := float64(todaySpan) / float64(span)
		todayCost := todayFrac * cost
		dateCost[today.Format("2006-01-02")] = todayCost

		today = todayEnd
	}

	return dateCost, nil
}

type DateTagValueCostTable map[[3]string]float64

func (table DateTagValueCostTable) WriteCSV(w io.Writer) error {
	csvw := csv.NewWriter(w)
	csvw.Write([]string{"Date", "Tag", "Value", "Cost"})
	for key, cost := range table {
		cols := append(key[:], fmt.Sprint(cost))
		err := csvw.Write(cols)
		if err != nil {
			return err
		}
	}
	csvw.Flush()
	return csvw.Error()
}

type AWSTag struct {
	name      string
	csvheader string
}

func GetAWSTags(header csvutil.Header) []*AWSTag {
	tags := make([]*AWSTag, 0, len(header)/2) // somewhat arbitrary cap
	// will not be sorted
	for colname := range header {
		tag, err := ParseAWSTag(colname)
		if err == nil {
			tags = append(tags, tag)
		}
	}
	return tags
}

const AWSTagPrefix = "user:"

func ParseAWSTag(csvheader string) (*AWSTag, error) {
	if strings.HasPrefix(csvheader, AWSTagPrefix) {
		tag := &AWSTag{
			name: csvheader[len(AWSTagPrefix):],
			csvheader: csvheader,
		}
		return tag, nil
	}
	return nil, fmt.Errorf("not a tag")
}
