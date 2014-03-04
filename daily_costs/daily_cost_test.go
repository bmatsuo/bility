package main

import (
	"github.com/bmatsuo/bility/go-awsbilling"
	"github.com/bmatsuo/bility/go-csvutil"

	"testing"
	"reflect"
)

func TestAWSDailyCostSingleDay(t *testing.T) {
	header := csvutil.Header{
		awsbilling.H_UsageStartDate: 0,
		awsbilling.H_UsageEndDate:   1,
		awsbilling.H_UnBlendedCost:  2,
	}

	costs, err := AWSDailyCost(header, []string{
		// 1 hour
		"2012-03-12 12:00:00",
		"2012-03-12 14:34:12",
		"60.00",
	})
	if err != nil {
		t.Fatalf("error computing daily cost: %v", err)
	}
	if len(costs) != 1 {
		t.Fatalf("expected number of cost entries: %d", len(costs))
	}

	costsExpected := map[string]float64{
		"2012-03-12": 60,
	}
	if !reflect.DeepEqual(costs, costsExpected) {
		t.Fatal("unexpected daily cost map: %v", costs)
	}
}

func TestAWSDailyCostMultiDay(t *testing.T) {
	header := csvutil.Header{
		awsbilling.H_UsageStartDate: 0,
		awsbilling.H_UsageEndDate:   1,
		awsbilling.H_UnBlendedCost:  2,
	}

	costs, err := AWSDailyCost(header, []string{
		// 48 hours split over 3 days.
		"2012-03-12 12:00:00",
		"2012-03-14 12:00:00",
		"48.00",
	})
	if err != nil {
		t.Fatalf("error computing daily cost: %v", err)
	}
	if len(costs) != 3 {
		t.Fatalf("expected number of cost entries: %d", len(costs))
	}

	costsExpected := map[string]float64{
		"2012-03-12": 12,
		"2012-03-13": 24,
		"2012-03-14": 12,
	}
	if !reflect.DeepEqual(costs, costsExpected) {
		t.Fatal("unexpected daily cost map: %v", costs)
	}
}
