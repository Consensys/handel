package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ConsenSys/handel/simul/platform"
	"github.com/stretchr/testify/require"
)

// This test runs the simulation with the `config_example.toml` config file
// and checks the output / return code
func TestMainLocalHost(t *testing.T) {
	resultsDir := "results"
	baseDir := "tests"
	configs := []string{"handel", "gossip", "udp"}
	for _, c := range configs {

		configName := c + ".toml"
		fullPath := filepath.Join(baseDir, configName)
		plat := "localhost"
		cmd := platform.NewCommand("go", "run", "main.go",
			"-config", fullPath,
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
		select {
		case out := <-foundCh:
			require.True(t, out)
		case <-time.After(1 * time.Minute):
			t.Fatalf("timeout in simulation " + configName)
		}
		require.FileExists(t, filepath.Join(resultsDir, c+".csv"))
	}
}
