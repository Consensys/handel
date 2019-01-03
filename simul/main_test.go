package main

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// This test runs the simulation with the `config_example.toml` config file
// and checks the output / return code
func TestMainLocalHost(t *testing.T) {
	configName := "config_example.toml"
	platform := "localhost"
	cmd := exec.Command("go", "run", "main.go",
		"-config", configName,
		"-platform", platform)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	require.Contains(t, string(out), "success !")
}
