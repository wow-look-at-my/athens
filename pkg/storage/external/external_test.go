package external

import (
	"net/http/httptest"
	"testing"

	"github.com/gomods/athens/pkg/storage/compliance"
	"github.com/gomods/athens/pkg/storage/mem"
	"github.com/stretchr/testify/require"
)

func TestExternal(t *testing.T) {
	strg, err := mem.NewStorage()
	require.Nil(t, err)

	handler := NewServer(strg)
	srv := httptest.NewServer(handler)
	defer srv.Close()
	externalStrg := NewClient(srv.URL, nil)
	clear := strg.(interface{ Clear() error }).Clear
	compliance.RunTests(t, externalStrg, clear)
}
