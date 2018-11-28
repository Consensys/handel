package monitor

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

type DummyCounter struct {
	rvalue float64
	wvalue float64
}

func (dm *DummyCounter) Values() map[string]float64 {
	dm.rvalue += 10
	dm.wvalue += 10
	return map[string]float64{
		"rvalue": float64(dm.rvalue),
		"wvalue": float64(dm.wvalue),
	}
}

func TestCounterMeasureRecord(t *testing.T) {
	mon, _ := setupMonitor(t)
	dm := &DummyCounter{0, 0}
	// create the counter measure
	cm := NewCounterMeasure("dummy", dm)
	if cm.baseMap["rvalue"] != dm.rvalue || cm.baseMap["wvalue"] != dm.wvalue {
		t.Logf("baseRx = %f vs rvalue = %f || baseTx = %f vs wvalue = %f", cm.baseMap["rvalue"], dm.rvalue, cm.baseMap["wvalue"], dm.wvalue)
		t.Fatal("Tx() / Rx() not working ?")
	}
	//bread, bwritten := cm.baseRx, cm.baseTx
	cm.Record()
	// check the values again
	if cm.baseMap["rvalue"] != dm.rvalue || cm.baseMap["wvalue"] != dm.wvalue {
		t.Fatal("Record() not working for CounterIOMeasure")
	}

	// Important otherwise data don't get written down to the monitor yet.
	time.Sleep(100 * time.Millisecond)
	str := new(bytes.Buffer)
	stat := mon.stats
	stat.Collect()
	stat.WriteHeader(str)
	stat.WriteValues(str)
	wr, re := stat.Value("dummy_wvalue"), stat.Value("dummy_rvalue")
	if wr == nil || wr.Avg() != 10.0 {
		t.Logf("stats => %v", stat.values)
		if wr != nil {
			t.Logf("wr.Avg() = %f", wr.Avg())
		}
		fmt.Printf("stats.values[dummy_rvalue] => %s\n", stat.values["dummy_rvalue"])
		fmt.Printf("stats.values[dummy_rvalue].Avg() => %v\n", stat.values["dummy_rvalue"].Avg() == 10.0)
		fmt.Printf("wr.String() %s ==> %v\n", wr, wr.Avg() == 10.0)
		t.Fatal("Stats doesn't have the right value (write)")
	}
	if re == nil || re.Avg() != 10 {
		t.Fatal("Stats doesn't have the right value (read)")
	}
	EndAndCleanup()
	time.Sleep(100 * time.Millisecond)
}
