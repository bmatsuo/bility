package main

import (
	"reflect"
	"testing"
	"time"
)

func TestTagChain(t *testing.T) {
	chain := newTagChain()
	trans := chain.Transitions()
	if len(trans) != 0 {
		t.Fatal("unexpected number of transitions: %d", len(trans))
	}
	baset := time.Now().Round(time.Second)
	timeat := func(n int) time.Time {
		return baset.Add(time.Duration(n) * time.Second)
	}
	t_1, t0, t1, t2 := timeat(-1), timeat(0), timeat(1), timeat(2)

	chain.Record(t0, "00") // append
	trans = chain.Transitions()
	expectTrans := []TagRecord{{t0, "00"}}
	if !reflect.DeepEqual(trans, expectTrans) {
		t.Fatal("unexpected number of transitions")
	}

	chain.Record(t2, "01") // append
	trans = chain.Transitions()
	expectTrans = []TagRecord{{t0, "00"}, {t2, "01"}}
	if !reflect.DeepEqual(trans, expectTrans) {
		t.Fatal("unexpected number of transitions")
	}

	chain.Record(t1, "00") // noop
	trans = chain.Transitions()
	// no change to expectTrans
	if !reflect.DeepEqual(trans, expectTrans) {
		t.Fatal("unexpected number of transitions")
	}

	chain.Record(t1, "01") // replace
	trans = chain.Transitions()
	expectTrans = []TagRecord{{t0, "00"}, {t1, "01"}}
	if !reflect.DeepEqual(trans, expectTrans) {
		t.Fatal("unexpected number of transitions")
	}

	chain.Record(t_1, "00") // replace
	trans = chain.Transitions()
	expectTrans = []TagRecord{{t_1, "00"}, {t1, "01"}}
	if !reflect.DeepEqual(trans, expectTrans) {
		t.Fatal("unexpected number of transitions")
	}
}
