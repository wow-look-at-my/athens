//go:build gcp

package stash

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/storage"
	"github.com/gomods/athens/pkg/storage/gcp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type failReader int

func (f *failReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("failure")
}

// TestWithGCS requires a real GCP backend implementation
// and it will ensure that saving to modules at the same time
// is done synchronously so that only the first module gets saved.
func TestWithGCS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	const (
		mod = "stashmod"
		ver = "v1.0.0"
	)
	strg := getStorage(t)
	strg.Delete(ctx, mod, ver)
	defer strg.Delete(ctx, mod, ver)

	// sanity check
	_, err := strg.GoMod(ctx, mod, ver)
	require.True(t, errors.Is(err, errors.KindNotFound))

	var eg errgroup.Group
	for i := 0; i < 5; i++ {
		content := uuid.New().String()
		ms := &mockGCPStasher{strg, content}
		gs, err := WithGCSLock(120, strg)
		require.Nil(t, err)

		s := gs(ms)
		eg.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			_, err := s.Stash(ctx, "stashmod", "v1.0.0")
			return err
		})
	}

	err = eg.Wait()
	require.Nil(t, err)

	info, err := strg.Info(ctx, mod, ver)
	require.Nil(t, err)

	modContent, err := strg.GoMod(ctx, mod, ver)
	require.Nil(t, err)

	zip, err := strg.Zip(ctx, mod, ver)
	require.Nil(t, err)

	defer zip.Close()
	zipContent, err := io.ReadAll(zip)
	require.Nil(t, err)

	require.True(t, bytes.Equal(info, modContent))

	require.True(t, bytes.Equal(info, zipContent))

}

// TestWithGCSPartialFailure equires a real GCP backend implementation
// and ensures that if one of the non-singleflight-lock files fails to
// upload, that the cache does not remain poisoned.
func TestWithGCSPartialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	const (
		mod = "stashmod"
		ver = "v1.0.0"
	)
	strg := getStorage(t)
	strg.Delete(ctx, mod, ver)
	defer strg.Delete(ctx, mod, ver)

	// sanity check
	_, err := strg.GoMod(ctx, mod, ver)
	require.True(t, errors.Is(err, errors.KindNotFound))

	content := uuid.New().String()
	ms := &mockGCPStasher{strg, content}
	fr := new(failReader)
	gs, err := WithGCSLock(120, strg)
	require.Nil(t, err)

	s := gs(ms)
	// We simulate a failure by manually passing an io.Reader that will fail.
	err = ms.strg.Save(ctx, "stashmod", "v1.0.0", []byte(ms.content), fr, nil, []byte(ms.content))
	require.NotNil(t, err)

	// Now try a Stash. This should upload the missing files.
	_, err = s.Stash(ctx, "stashmod", "v1.0.0")
	require.Nil(t, err)

	info, err := strg.Info(ctx, mod, ver)
	require.Nil(t, err)

	modContent, err := strg.GoMod(ctx, mod, ver)
	require.Nil(t, err)

	zip, err := strg.Zip(ctx, mod, ver)
	require.Nil(t, err)

	defer zip.Close()
	zipContent, err := io.ReadAll(zip)
	require.Nil(t, err)

	require.True(t, bytes.Equal(info, modContent))

	require.True(t, bytes.Equal(info, zipContent))

}

// mockGCPStasher is like mockStasher
// but leverages in memory storage
// so that redis can determine
// whether to call the underlying stasher or not.
type mockGCPStasher struct {
	strg    storage.Backend
	content string
}

func (ms *mockGCPStasher) Stash(ctx context.Context, mod, ver string) (string, error) {
	err := ms.strg.Save(
		ctx,
		mod,
		ver,
		[]byte(ms.content),
		strings.NewReader(ms.content),
		nil,
		[]byte(ms.content),
	)
	return "", err
}

func getStorage(t *testing.T) *gcp.Storage {
	t.Helper()
	cfg := getTestConfig()
	if cfg == nil {
		t.SkipNow()
	}

	s, err := gcp.New(context.Background(), cfg, config.GetTimeoutDuration(30))
	require.Nil(t, err)

	return s
}

func getTestConfig() *config.GCPConfig {
	creds := os.Getenv("GCS_SERVICE_ACCOUNT")
	if creds == "" {
		return nil
	}
	return &config.GCPConfig{
		Bucket:  "athens_drone_stash_bucket",
		JSONKey: creds,
	}
}
