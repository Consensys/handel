package utils

import (
	"bufio"
	"encoding/csv"
	"os"

	h "github.com/ConsenSys/handel"
)

// NodeRecord represents line of CSV file and
// implements the Identity inteface
type NodeRecord struct {
	id     int32
	port   int
	addr   string
	pubKey string
}

func (identity NodeRecord) Address() string {
	return identity.addr
}

// TODO agree on PublicKey serialization format
func (identity NodeRecord) PublicKey() h.PublicKey {
	return nil
}

func (identity NodeRecord) ID() int32 {
	return identity.id
}

// CsvConstructor crates an NodeRecord based on csv file record
type CsvConstructor interface {
	Read(line []string) (*NodeRecord, error)
}

func readCSVFile(path string) [][]string {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(file)
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}
	return records
}

// Properly formatted csv file has the following header
// id, port, addr, pubKey
func makeRegistry(records [][]string, localID int32, lineParser CsvConstructor) (h.Registry, int, error) {
	listenPort := 0
	nodes := []h.Identity{}
	//start from index 1 , skip the header
	for i := 1; i < len(records); i++ {
		rec := records[i]

		nodeRecord, err := lineParser.Read(rec)
		if err != nil {
			return nil, -1, err
		}
		if nodeRecord.ID() == localID {
			listenPort = nodeRecord.port
		}
		nodes = append(nodes, nodeRecord)
	}
	return h.NewArrayRegistry(nodes), listenPort, nil
}

// ReadCSVRegistry creates a Registry based on provided CSV file.
// Returns the registry and listen port of the local peer
func ReadCSVRegistry(path string, localID int32, lineParser CsvConstructor) (h.Registry, int) {
	records := readCSVFile(path)
	reg, port, err := makeRegistry(records, localID, lineParser)
	if err != nil {
		panic(err)
	}
	return reg, port
}
