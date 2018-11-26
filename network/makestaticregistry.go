package network

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	h "github.com/ConsenSys/handel"
)

type nodeRecord struct {
	id     int32
	port   int
	addr   string
	pubKey string
}

func (identity nodeRecord) Address() string {
	return identity.addr
}

//TODO agree on PublicKey serialization format
func (identity nodeRecord) PublicKey() h.PublicKey {
	return nil
}

func (identity nodeRecord) ID() int32 {
	return identity.id
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
func makeRegistry(records [][]string, localID int32) (h.Registry, int, error) {
	listenPort := 0
	nodes := []h.Identity{}
	//start from index 1 , skip the header
	for i := 1; i < len(records); i++ {
		rec := records[i]

		i, err := strconv.ParseInt(rec[0], 10, 32)
		if err != nil {
			return nil, -1, err
		}
		id := int32(i)

		p, err := strconv.ParseInt(rec[1], 10, 32)
		if err != nil {
			return nil, -1, err
		}
		pt := int(p)
		if id == localID {
			listenPort = pt
		}
		addr := fmt.Sprintf("%s:%d", rec[2], pt)

		//TODO read peer's serialized public key
		node := nodeRecord{id: id, port: pt, addr: addr, pubKey: ""}
		var identity h.Identity = node
		nodes = append(nodes, identity)
	}
	return h.NewArrayRegistry(nodes), listenPort, nil
}

// ReadCSV creates a Registry based on provided CSV file.
// Returns the registry and listen port of the local peer/
func ReadCSV(path string, localID int32) (h.Registry, int) {
	records := readCSVFile(path)
	reg, port, err := makeRegistry(records, localID)
	if err != nil {
		panic(err)
	}
	return reg, port
}
