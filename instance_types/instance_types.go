package main

import (
	"github.com/bmatsuo/bility/go-csvutil"
	"encoding/csv"
	"fmt"
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

	header, err := csvutil.ReadHeader(r)
	if err != nil {
		log.Fatalf("failed reading csv header; %v", err)
	}

	// process rows as a stream
	rowch := csvutil.NewStream(r, 1)
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

func GetBoxType(header csvutil.Header, cols []string) (string, error) {
	for _, filter := range []struct{ col, value string }{
		{"ProductName", "Amazon Elastic Compute Cloud"},
		{"Operation", "RunInstances"},
	} {
		val, err := csvutil.GetColumn(header, cols, filter.col)
		if err != nil {
			return "", err
		}
		if val != filter.value {
			return "", BadRowTypeErr
		}
	}

	var boxtype string
	boxtypePrefix := "BoxUsage"
	utype, err := csvutil.GetColumn(header, cols, "UsageType")
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
