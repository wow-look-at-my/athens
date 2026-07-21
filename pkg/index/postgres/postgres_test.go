package postgres

import (
	"os"
	"testing"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/index/compliance"
	"github.com/stretchr/testify/require"
)

func TestPostgres(t *testing.T) {
	if os.Getenv("TEST_INDEX_POSTGRES") != "true" {
		t.SkipNow()
	}
	cfg := getTestConfig(t)
	i, err := New(cfg)
	require.Nil(t, err)

	compliance.RunTests(t, i, i.(*indexer).clear)
}

func (i *indexer) clear() error {
	_, err := i.db.Exec(`DELETE FROM indexes`)
	return err
}

func getTestConfig(t *testing.T) *config.Postgres {
	t.Helper()
	cfg, err := config.Load("")
	require.Nil(t, err)

	return cfg.Index.Postgres
}
