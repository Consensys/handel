package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"sync"

	"go.dedis.ch/onet/v3/log"
)

var global struct {
	// Sink is the server address where all measures are transmitted to for
	// further analysis.
	sink string

	// Structs are encoded through a json encoder.
	encoder *json.Encoder
	conn    *net.UDPConn

	sync.Mutex
}

// Measure is an interface for measurements
// Usage:
// 		measure := monitor.SingleMeasure("bandwidth")
// or
//		measure := monitor.NewTimeMeasure("round")
// 		measure.Record()
type Measure interface {
	// Record must be called when you want to send the value
	// over the monitor listening.
	// Implementation of this interface must RESET the value to `0` at the end
	// of Record(). `0` means the initial value / meaning this measure had when
	// created.
	// Example: TimeMeasure.Record() will reset the time to `time.Now()`
	//          CounterMeasure.Record() will  reset the counter of the bytes
	//          read / written to 0.
	//          etc
	Record()
}

// SingleMeasure is a pair name - value we want to send to the monitor.
type singleMeasure struct {
	Name  string
	Value float64
}

// TimeMeasure represents a measure regarding time: It includes the wallclock
// time, the cpu time + the user time.
type TimeMeasure struct {
	Wall *singleMeasure
	CPU  *singleMeasure
	User *singleMeasure
	// non exported fields
	// name of the time measure (basename)
	name string
	// last time
	lastWallTime time.Time
}

// ConnectSink connects to the given endpoint and initialises a json
// encoder. It can be the address of a proxy or a monitoring process.
// Returns an error if it could not connect to the endpoint.
func ConnectSink(addr string) error {
	global.Lock()
	defer global.Unlock()
	if global.conn != nil {
		return errors.New("already connected to an endpoint")
	}
	log.Lvl3("Connecting to:", addr)
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	global.sink = addr
	global.conn = conn
	global.encoder = json.NewEncoder(conn)
	return nil
}

// RecordSingleMeasure sends the pair name - value to the monitor directly.
func RecordSingleMeasure(name string, value float64) {
	sm := newSingleMeasure(name, value)
	sm.Record()
}

func newSingleMeasure(name string, value float64) *singleMeasure {
	return &singleMeasure{
		Name:  name,
		Value: value,
	}
}

func (s *singleMeasure) Record() {
	if err := send(s); err != nil {
		log.Error("Error sending SingleMeasure", s.Name, " to monitor:", err)
	}
}

// NewTimeMeasure return *TimeMeasure
func NewTimeMeasure(name string) *TimeMeasure {
	tm := &TimeMeasure{name: name}
	tm.reset()
	return tm
}

// Record sends the measurements to the monitor:
//
// - wall time: *name*_wall
//
// - system time: *name*_system
//
// - user time: *name*_user
func (tm *TimeMeasure) Record() {
	// Wall time measurement
	tm.Wall = newSingleMeasure(tm.name+"_wall", float64(time.Since(tm.lastWallTime))/1.0e9)
	// CPU time measurement
	tm.CPU.Value, tm.User.Value = getDiffRTime(tm.CPU.Value, tm.User.Value)
	// send data
	tm.Wall.Record()
	tm.CPU.Record()
	tm.User.Record()
	// reset timers
	tm.reset()

}

// reset reset the time fields of this time measure
func (tm *TimeMeasure) reset() {
	cpuTimeSys, cpuTimeUser := getRTime()
	tm.CPU = newSingleMeasure(tm.name+"_system", cpuTimeSys)
	tm.User = newSingleMeasure(tm.name+"_user", cpuTimeUser)
	tm.lastWallTime = time.Now()
}

// Counter is an interface that can be used to report multiple values that
// keeps evolving. The keys in the returned map is the name of the value to
// record.
type Counter interface {
	Values() map[string]float64
}

// CounterMeasure is a struct that takes a Counter and can send the
// measurements to the monitor. Each time Record() is called, the measurements
// are put back to 0 (while the Counter still sends increased bytes number).
type CounterMeasure struct {
	name    string
	counter Counter
	baseMap map[string]float64
}

// NewCounterMeasure returns an CounterMeasure fresh. The base value are set to
// the values returned by counter.Values().
func NewCounterMeasure(name string, counter Counter) *CounterMeasure {
	return &CounterMeasure{
		name:    name,
		counter: counter,
		baseMap: counter.Values(),
	}
}

// Record send the actual number of bytes read and written (**name**_written &
// **name**_read) and reset the counters.
func (cm *CounterMeasure) Record() {
	newMap := cm.counter.Values()
	for k, v := range cm.baseMap {
		newV, ok := newMap[k]
		if !ok {
			continue
		}
		diff := newV - v
		measure := newSingleMeasure(cm.name+"_"+k, diff)
		measure.Record()
		cm.baseMap[k] = newV
	}
}

// Send transmits the given struct over the network.
func send(v interface{}) error {
	global.Lock()
	defer global.Unlock()
	if global.conn == nil {
		return fmt.Errorf("monitor's sink connection not initialized")
	}
	// For a large number of clients (˜10'000), the connection phase
	// can take some time. This is a linear backoff to enable connection
	// even when there are a lot of request:
	var ok bool
	var err error
	for wait := 500; wait < 1000; wait += 100 {
		if err = global.encoder.Encode(v); err == nil {
			ok = true
			break
		}
		fmt.Println("message NOT sent", err)
		time.Sleep(time.Duration(wait) * time.Millisecond)
		continue
	}
	if !ok {
		return errors.New("could not send any measures")
	}
	return nil
}

// EndAndCleanup sends a message to end the logging and closes the connection
func EndAndCleanup() {
	global.Lock()
	defer global.Unlock()
	if err := global.conn.Close(); err != nil {
		// at least tell that we could not close the connection:
		log.Error("Could not close connection:", err)
	}
	global.conn = nil
}

// Returns the difference of the given system- and user-time.
func getDiffRTime(tSys, tUsr float64) (tDiffSys, tDiffUsr float64) {
	nowSys, nowUsr := getRTime()
	return nowSys - tSys, nowUsr - tUsr
}
