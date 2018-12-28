package lib

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func csvContent() [][]string {
	csv := [][]string{
		{"0", "127.0.0.1:3000", "aed142", "aed142"},
		{"1", "127.0.0.1:3001", "aed142", "aed142"},
		{"2", "127.0.0.1:3002", "aed142", "aed142"},
	}
	return csv
}

func csvCorrupted() [][]string {
	csv := [][]string{
		{"0", "127.0.0.1:3000", "", ""},
		{"x", "127.0.0.1:3001", "", ""},
		{"2", "127.0.0.1:3002", "", ""},
	}
	return csv
}

func writeCSV(records [][]string) string {
	file, err := ioutil.TempFile("/tmp", "*")
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(file)
	for _, record := range records {
		if err := w.Write(record); err != nil {
			panic(err)
		}
	}
	w.Flush()
	name := file.Name()
	file.Close()
	return name
}

func TestCSVParser(t *testing.T) {
	parser := NewCSVParser()
	cons := NewEmptyConstructor()

	type csvTest struct {
		csv     [][]string
		expErr  bool
		idx     int32
		expAddr string
		expID   int32
		expSize int
		reqID   int32
		reqAddr string
	}

	var tests = []csvTest{
		{csvContent(), false, 1, "127.0.0.1:3001", 1, 3, 2, "127.0.0.1:3002"},
		{csv: csvCorrupted(), expErr: true},
	}

	for i, test := range tests {
		fmt.Printf(" --- test %d ---\n", i)
		name := writeCSV(test.csv)
		defer os.RemoveAll(name)

		reg, node, err := ReadAll(name, 1, parser, cons)
		if test.expErr {
			require.Error(t, err)
			continue
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, test.expAddr, node.Identity.Address())
		require.Equal(t, test.expID, node.Identity.ID())
		require.Equal(t, test.expSize, reg.Size())
		id, _ := reg.Identity(int(test.reqID))
		require.Equal(t, test.reqAddr, id.Address())
	}

}
