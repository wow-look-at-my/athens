package stash

import (
	"testing"

	"github.com/gomods/athens/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGetRedisClientOptions_HostPort(t *testing.T) {
	opts, err := getRedisClientOptions("localhost:6379", "mypassword")
	require.NoError(t, err)
	require.Equal(t, "localhost:6379", opts.Addr)
	require.Equal(t, "mypassword", opts.Password)
	require.Equal(t, "tcp", opts.Network)
}

func TestGetRedisClientOptions_URL(t *testing.T) {
	opts, err := getRedisClientOptions("redis://user:secret@localhost:6379/0", "secret")
	require.NoError(t, err)
	require.Equal(t, "localhost:6379", opts.Addr)
	require.Equal(t, "secret", opts.Password)
}

func TestGetRedisClientOptions_URL_NoPassword(t *testing.T) {
	opts, err := getRedisClientOptions("redis://localhost:6379/0", "")
	require.NoError(t, err)
	require.Equal(t, "localhost:6379", opts.Addr)
}

func TestGetRedisClientOptions_URL_PasswordMismatch(t *testing.T) {
	_, err := getRedisClientOptions("redis://user:urlpassword@localhost:6379/0", "differentpassword")
	require.Error(t, err)
	require.ErrorIs(t, err, errPasswordsDoNotMatch)
}

func TestLockOptionsFromConfig_Valid(t *testing.T) {
	lc := &config.RedisLockConfig{
		TTL:        30,
		Timeout:    5,
		MaxRetries: 3,
	}
	opts, err := lockOptionsFromConfig(lc)
	require.NoError(t, err)
	require.Equal(t, 3, opts.maxRetries)
}

func TestLockOptionsFromConfig_InvalidTTL(t *testing.T) {
	lc := &config.RedisLockConfig{
		TTL:        0,
		Timeout:    5,
		MaxRetries: 3,
	}
	_, err := lockOptionsFromConfig(lc)
	require.Error(t, err)
}

func TestLockOptionsFromConfig_InvalidTimeout(t *testing.T) {
	lc := &config.RedisLockConfig{
		TTL:        30,
		Timeout:    -1,
		MaxRetries: 3,
	}
	_, err := lockOptionsFromConfig(lc)
	require.Error(t, err)
}

func TestLockOptionsFromConfig_InvalidMaxRetries(t *testing.T) {
	lc := &config.RedisLockConfig{
		TTL:        30,
		Timeout:    5,
		MaxRetries: 0,
	}
	_, err := lockOptionsFromConfig(lc)
	require.Error(t, err)
}
