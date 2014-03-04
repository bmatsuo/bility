package main

import (
	"github.com/bmatsuo/bility/go-awsbilling"
	"github.com/bmatsuo/bility/go-csvutil"

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


	header, stream, err := awsbilling.NewCSVStream(file, 1)
	if err != nil {
		log.Fatalf("failed opening data stream; %v", err)
	}

	seen := make(map[string]bool, 0)

	for row := range stream {
		if row.Err != nil {
			log.Fatalf("failed reading csv data; %v", row.Err)
		}

		isEC2, _ := awsbilling.IsEC2InstanceBillingItem(header, row.Cols)
		if !isEC2 {
			continue
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
