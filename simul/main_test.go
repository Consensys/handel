package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ConsenSys/handel/simul/platform"
	"github.com/stretchr/testify/require"
)

// This test runs the simulation with the `config_example.toml` config file
// and checks the output / return code
func TestMainLocalHost(t *testing.T) {
	resultsDir := "results"
	configName := "config_example.toml"
	plat := "localhost"
	cmd := platform.NewCommand("go", "run", "main.go",
		"-config", configName,
		"-platform", plat)
	chLine := cmd.LineOutput()
	foundCh := make(chan bool, 1)
	go func() {
		found := false
		for line := range chLine {
			fmt.Println(line)
			if strings.Contains(line, "success") {
				foundCh <- true
				found = true
			}
		}
		if !found {
			foundCh <- false
		}
	}()
	err := cmd.Cmd.Run()
	require.NoError(t, err)

	require.True(t, <-foundCh)

	require.FileExists(t, filepath.Join(resultsDir, "config_example.csv"))
}
