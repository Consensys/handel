package lib

import (
	"bufio"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"github.com/ConsenSys/handel"
)

// NodeParser is an interface that can read / write node records.
type NodeParser interface {
	// Reads all NodeRecords  from a given URI. It can be a CSV file
	// encoded for example.
	Read(uri string) ([]*NodeRecord, error)
	// Writes all node records to an URI. It can be a file.
	Write(uri string, records []*NodeRecord) error
}

// NodeList is a type that contains all informations on all nodes, and that
// implements the Registry interface. It is useful for binaries that retrieves
// multiple node information - not only the Identity.
type NodeList []*Node

// Size implements the handel.Registry interface
func (n *NodeList) Size() int {
	return len(*n)
}

// Identity implements the handel.Registry interface
func (n *NodeList) Identity(idx int) (handel.Identity, bool) {
	if idx < 0 || idx >= n.Size() {
		return nil, false
	}
	return (*n)[idx], true
}

// Registry returns a handel.Registry interface
func (n *NodeList) Registry() handel.Registry {
	ids := make([]handel.Identity, len(*n))
	for i := 0; i < len(ids); i++ {
		id := (*n)[i]
		idx := int(id.ID())
		ids[idx] = id
	}
	return handel.NewArrayRegistry(ids)
}

// Identities implements the handel.Registry interface
func (n *NodeList) Identities(from, to int) ([]handel.Identity, bool) {
	if !n.inBound(from) || !n.inBound(to) {
		return nil, false
	}
	if to < from {
		return nil, false
	}
	nodes := (*n)[from:to]
	ids := make([]handel.Identity, len(nodes))
	for i, n := range nodes {
		ids[i] = n.Identity
	}
	return ids, true
}

func (n *NodeList) inBound(idx int) bool {
	return !(idx < 0 || idx > len(*n))
}

// Node returns the Node structure at the given index
func (n *NodeList) Node(i int) *Node {
	if i < 0 || i > len(*n) {
		panic("that should not happen")
	}
	return (*n)[i]
}

// ReadAll reads the whole set of nodes from the given parser to the given URI.
// It returns the node list which can be used as a Registry as well
func ReadAll(uri string, parser NodeParser, c Constructor) (NodeList, error) {
	records, err := parser.Read(uri)
	if err != nil {
		return nil, err
	}
	var nodes = make([]*Node, len(records))
	for _, rec := range records {
		node, err := rec.ToNode(c)
		if err != nil {
			return nil, err
		}
		nodes[int(node.ID())] = node
	}
	return nodes, nil
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
	csvReader.FieldsPerRecord = 5
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
		isByz, err := strconv.ParseBool(line[4])
		if err != nil {
			return nil, err
		}
		nodeRecord := &NodeRecord{ID: id, Addr: addr, Private: priv, Public: pub, IsByzantine: isByz}
		nodes = append(nodes, nodeRecord)
	}
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
			record.Public,
			strconv.FormatBool(record.IsByzantine)}
		if err := w.Write(line); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}
