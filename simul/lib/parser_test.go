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
	csvf := [][]string{
		{"0", "127.0.0.1:3000", "aed142", "aed142", "false"},
		{"1", "127.0.0.1:3001", "aed142", "aed142", "true"},
		{"2", "127.0.0.1:3002", "aed142", "aed142", "false"},
	}
	return csvf
}

func csvCorrupted() [][]string {
	csvf := [][]string{
		{"0", "127.0.0.1:3000", "", "", "false"},
		{"x", "127.0.0.1:3001", "", "", "false"},
		{"2", "127.0.0.1:3002", "", "", "true"},
	}
	return csvf
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
		expSize int
	}

	var tests = []csvTest{
		{csv: csvContent(), expErr: false, expSize: 3},
		{csv: csvCorrupted(), expErr: true},
	}

	for i, test := range tests {
		fmt.Printf(" --- test %d ---\n", i)
		name := writeCSV(test.csv)
		defer os.RemoveAll(name)

		nodeList, err := ReadAll(name, parser, cons)
		if test.expErr {
			require.Error(t, err)
			continue
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, test.expSize, nodeList.Registry().Size())
	}

}
