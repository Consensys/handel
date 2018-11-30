package lib

import (
	"bufio"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	h "github.com/ConsenSys/handel"
)

// NodeParser is an interface that can read / write node records.
type NodeParser interface {
	// Reads all NodeRecords  from a given URI. It can be a CSV file
	// encoded for example.
	Read(uri string) ([]*NodeRecord, error)
	// Writes all node records to an URI. It can be a file.
	Write(uri string, records []*NodeRecord) error
}

// ReadAll reads the whole set of nodes from the given parser to the given URI.
// It returns the registry and the node corresponding to the ID given.
func ReadAll(uri string, id int, parser NodeParser, c Constructor) (h.Registry, *Node, error) {
	var ids []h.Identity
	var ownNode *Node
	records, err := parser.Read(uri)
	if err != nil {
		return nil, nil, err
	}
	for _, rec := range records {
		node, err := rec.ToNode(c)
		if err != nil {
			return nil, nil, err
		}
		if rec.ID == int32(id) {
			ownNode = node
		}
		ids = append(ids, node.Identity)
	}
	return h.NewArrayRegistry(ids), ownNode, nil
}

type csvParser struct{}

// NewCSVParser is a NodeParser that reads/writes to a CSV file
func NewCSVParser() NodeParser {
	return &csvParser{}
}

// Read implements NodeParser
func (c *csvParser) Read(uri string) ([]*NodeRecord, error) {
	file, err := os.Open(uri)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(file)
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = 4
	var nodes []*NodeRecord
	for {
		line, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return nodes, nil
			}
			return nil, err
		}

		i, err := strconv.ParseInt(line[0], 10, 32)
		if err != nil {
			return nil, err
		}
		id := int32(i)
		addr := line[1]
		priv := line[2]
		pub := line[3]
		nodeRecord := &NodeRecord{ID: id, Addr: addr, Private: priv, Public: pub}
		nodes = append(nodes, nodeRecord)
	}
	return nodes, nil
}

func (c *csvParser) Write(uri string, records []*NodeRecord) error {
	file, err := os.Create(uri)
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	for _, record := range records {
		line := []string{strconv.Itoa(int(record.ID)),
			record.Addr,
			record.Private,
			record.Public}
		if err := w.Write(line); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}
