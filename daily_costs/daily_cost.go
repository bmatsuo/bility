// rows which are missing a column for a tag are aggregated as if the column has
// an empty string value. It's not entirely clear if this is correct. 
package main

import (
	"github.com/bmatsuo/bility/go-awsbilling"
	"github.com/bmatsuo/bility/go-csvutil"

	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
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

	header, stream, err := awsbilling.NewCSVStream(file, 1)
	if err != nil {
		log.Fatalf("failed opening csv stream; %v", err)
	}

	costs := make(DateTagValueCostTable)

	tags := awsbilling.GetTags(header)
	if len(tags) == 0 {
		log.Print("no tags present in header")
		dumpAndDie(costs)
	}

	for row := range stream {
		if row.Err != nil {
			log.Fatalf("failed reading csv data; %v", row.Err)
		}

		isSummary, _ := awsbilling.IsSummaryItem(header, row.Cols)
		if isSummary {
			continue
		}

		dailyCost, err := AWSDailyCost(header, row.Cols)
		if err != nil {
			log.Printf("error reading cost: %v (%v)", err, row.Cols)
			continue
		}
		if len(dailyCost) == 0 {
			continue
		}

		for _, tag := range tags {
			// errors (missing columns) are treated as empty strings
			val, _ := csvutil.GetColumn(header, row.Cols, tag.CSVHeader())
			for date, cost := range dailyCost {
				key := [3]string{date, tag.Name(), val}
				costs[key] += cost
			}
		}
	}

	dumpAndDie(costs)
}

func dumpAndDie(costs DateTagValueCostTable) {
	// dumps cost summary as csv and exits when called
	err := costs.WriteCSV(os.Stdout)
	if err != nil {
		log.Fatal("output error: ", err)
	}
	os.Exit(0)
}

func AWSDailyCost(header csvutil.Header, cols []string) (map[string]float64, error) {
	_cost, err := csvutil.GetColumn(header, cols, awsbilling.H_UnBlendedCost)
	if err != nil || _cost == "" {
		return nil, nil
	}
	cost, err := strconv.ParseFloat(_cost, 64)
	if err != nil {
		return nil, fmt.Errorf("coludn't parse cost %q: %v", _cost, err)
	}

	_start, err := csvutil.GetColumn(header, cols, awsbilling.H_UsageStartDate)
	if err != nil || _start == "" {
		return nil, nil
	}
	start, err := awsbilling.ParseTime(_start)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse start time %q: %v", _start, err)
	}

	_end, err := csvutil.GetColumn(header, cols, awsbilling.H_UsageEndDate)
	if err != nil || _end == "" {
		return nil, nil
	}
	end, err := awsbilling.ParseTime(_end)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse end time %q: %v", _end, err)
	}

	span := end.Sub(start)
	if span < 0 {
		return nil, fmt.Errorf("start time after end time")
	}

	dateCost := make(map[string]float64)
	today := start
	for done := false; !done; {
		todayEnd := today.Round(24 * time.Hour)
		if todayEnd.Before(today) || todayEnd.Equal(today) {
			todayEnd = todayEnd.Add(24 * time.Hour)
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

// slices are not hashable
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
