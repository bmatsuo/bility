package main

import (
	//"github.com/bmatsuo/csv"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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

	header, err := ReadHeader(r)
	if err != nil {
		log.Fatalf("failed reading csv header; %v", err)
	}

	// process rows as a stream
	rowch := CSVRowStream(r, 1)
	seen := make(map[string]bool, 0)
	for row := range rowch {
		if row.Err != nil {
			log.Fatalf("failed reading csv data; %v", row.Err)
		}

		boxtype, err := GetBoxType(header, row.Cols)
		if err != nil {
			continue
		}
		seen[boxtype] = true
	}

	fmt.Println(len(seen))
	for k := range seen {
		fmt.Println(k)
	}
}

var BadRowTypeErr = fmt.Errorf("bad row type")

func GetBoxType(header CSVHeader, cols []string) (string, error) {
	for _, filter := range []struct{ col, value string }{
		{"ProductName", "Amazon Elastic Compute Cloud"},
		{"Operation", "RunInstances"},
	} {
		val, err := GetColumn(header, cols, filter.col)
		if err != nil {
			return "", err
		}
		if val != filter.value {
			return "", BadRowTypeErr
		}
	}

	var boxtype string
	boxtypePrefix := "BoxUsage"
	utype, err := GetColumn(header, cols, "UsageType")
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(utype, boxtypePrefix) {
		boxtype = utype[len(boxtypePrefix):]
	} else {
		return "", BadRowTypeErr
	}
	if boxtype == "" {
		boxtype = "m1.small"
	} else if boxtype[0] == ':' {
		boxtype = boxtype[1:]
	} else {
		log.Printf("suspicious ec2 instance usage-type encountered; %q", utype)
		return "", BadRowTypeErr
	}

	return boxtype, nil
}

var UnknownColumnErr = fmt.Errorf("unknown column")
var MissingColumnErr = fmt.Errorf("missing column")

// Returns UnknownColumnErr if name does not exist in header.
// Returns MissingColumnErr if cols does not contain a name column (not enough
// columns).
func GetColumn(header CSVHeader, cols []string, name string) (string, error) {
	i, ok := header[name]
	if !ok {
		return "", UnknownColumnErr
	}
	if i >= len(cols) {
		return "", MissingColumnErr
	}
	return cols[i], nil
}

type CSVHeader map[string]int

func ReadHeader(r *csv.Reader) (CSVHeader, error) {
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

type CSVRow struct {
	Cols []string
	Err  error
}

// read from r asynchronously.
func CSVRowStream(r *csv.Reader, bufsize uint) <-chan *CSVRow {
	ch := make(chan *CSVRow, bufsize)
	go func() {
		defer close(ch)
		for {
			cols, err := r.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Print("err ", err)
				ch <- &CSVRow{nil, err}
				return
			}
			ch <- &CSVRow{cols, nil}
		}
		panic("unreachable")
	}()
	return ch
}
