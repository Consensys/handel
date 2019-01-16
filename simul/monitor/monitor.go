// Package monitor package handle the logging, collection and computation of
// statistical data. Every application can send some Measure (for the moment,
// we mostly measure the CPU time but it can be applied later for any kind of
// measures). The Monitor receives them and updates a Stats struct. This Stats
// struct can hold many different kinds of Measurements (the measure of a
// specific action such as "round time" or "verify time" etc). These
// measurements contain Values which compute the actual min/max/dev/avg values.
//
// The Proxy allows to relay Measure from
// clients to the listening Monitor. A starter feature is also the DataFilter
// which can apply some filtering rules to the data before making any
// statistics about them.
package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/dedis/onet/log"
)

// This file handles the collection of measurements, aggregates them and
// write CSV file reports

// Sink is the address where to listen for the monitor. The endpoint can be a
// monitor.Proxy or a direct connection with measure.go
const Sink = "0.0.0.0"

// DefaultSinkPort is the default port where a monitor will listen and a proxy
// will contact the monitor.
const DefaultSinkPort = 10000

// Monitor struct is used to collect measures and make the statistics about
// them. It takes a stats object so it update that in a concurrent-safe manner
// for each new measure it receives.
type Monitor struct {
	sync.Mutex

	sock *net.UDPConn

	// Current stats
	stats *Stats

	// channel to give new measures
	measures chan *singleMeasure

	// channel to notify the end of a connection
	// send the name of the connection when finishd
	done chan string

	sinkPort     uint16
	sinkPortChan chan uint16
}

// NewDefaultMonitor returns a new monitor given the stats
func NewDefaultMonitor(stats *Stats) *Monitor {
	return &Monitor{
		stats:    stats,
		sinkPort: DefaultSinkPort,
		measures: make(chan *singleMeasure),
		done:     make(chan string),
	}
}

// NewMonitor returns a monitor listening on the given port
func NewMonitor(port int, stats *Stats) *Monitor {
	m := NewDefaultMonitor(stats)
	m.sinkPort = uint16(port)
	return m
}

// Listen will start listening for incoming connections on this address
// It needs the stats struct pointer to update when measures come
// Return an error if something went wrong during the connection setup
func (m *Monitor) Listen() error {
	addr := net.JoinHostPort(Sink, strconv.Itoa(int(m.sinkPort)))
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	udpSock, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return fmt.Errorf("Error while monitor is binding address: %v", err)
	}
	m.Lock()
	m.sock = udpSock
	m.Unlock()
	go m.handleConnection()
	log.Lvl2("Monitor listening for stats on", Sink, ":", m.sinkPort)
	<-m.done
	return nil
}

// Stop will close every connections it has
// And will stop updating the stats
func (m *Monitor) Stop() {
	log.Lvl2("Monitor Stop")
	m.Lock()
	if m.sock != nil {
		if err := m.sock.Close(); err != nil {
			fmt.Println("error closing: ", err)
		}
	}
	close(m.done)
	m.Unlock()
}

// handleConnection will decode the data received and aggregates it into its
// stats
func (m *Monitor) handleConnection() {
	nerr := 0
	reader := bufio.NewReader(m.sock)
	dec := json.NewDecoder(reader)
	for {
		select {
		case _, d := <-m.done:
			if !d {
				fmt.Println("leaving udp handling")
				return
			}
		default:
		}

		measure := &singleMeasure{}
		if err := dec.Decode(measure); err != nil {
			// if end of connection
			if strings.Contains(err.Error(), "closed") {
				break
			}
			nerr++
			if nerr > 50 {
				break
			}
		}

		// Special case where the measurement is indicating a FINISHED step
		switch strings.ToLower(measure.Name) {
		case "end":
			break
		default:
			m.update(measure)
		}
	}
}

// updateMeasures will add that specific measure to the global stats
// in a concurrently safe manner
func (m *Monitor) update(meas *singleMeasure) {
	// updating
	m.stats.Update(meas)
}
