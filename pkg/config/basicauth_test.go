package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicAuth_Both(t *testing.T) {
	c := &Config{BasicAuthUser: "user", BasicAuthPass: "pass"}
	user, pass, ok := c.BasicAuth()
	require.True(t, ok)
	require.Equal(t, "user", user)
	require.Equal(t, "pass", pass)
}

func TestBasicAuth_NoUser(t *testing.T) {
	c := &Config{BasicAuthPass: "pass"}
	_, _, ok := c.BasicAuth()
	require.False(t, ok)
}

func TestBasicAuth_NoPass(t *testing.T) {
	c := &Config{BasicAuthUser: "user"}
	_, _, ok := c.BasicAuth()
	require.False(t, ok)
}

func TestBasicAuth_Empty(t *testing.T) {
	c := &Config{}
	_, _, ok := c.BasicAuth()
	require.False(t, ok)
}

func TestFilterOff_Empty(t *testing.T) {
	c := &Config{}
	require.True(t, c.FilterOff())
}

func TestFilterOff_WithFile(t *testing.T) {
	c := &Config{FilterFile: "/path/to/filter"}
	require.False(t, c.FilterOff())
}

func TestLoad_Default(t *testing.T) {
	// Ensure no athens.toml exists in cwd
	orig, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	defer os.Chdir(orig)

	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "memory", cfg.StorageType)
	require.Equal(t, ":3000", cfg.Port)
}

func TestLoad_ExplicitFile(t *testing.T) {
	configFile := "../../config.dev.toml"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Skip("config.dev.toml not found")
	}
	cfg, err := Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.toml")
	require.Error(t, err)
}

func TestTimeoutDuration(t *testing.T) {
	c := &Config{TimeoutConf: TimeoutConf{Timeout: 300}}
	d := c.TimeoutDuration()
	require.Equal(t, 300, int(d.Seconds()))
}
