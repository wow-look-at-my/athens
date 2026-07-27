package actions

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSumdbProxy(t *testing.T) {
	var givenURL string
	expectedURL := "/latest"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		givenURL = r.URL.Path
	}))
	defer s.Close()

	surl, err := url.Parse(s.URL)
	if err != nil {
		panic(err)
	}
	pathPrefix := "/sumdb/" + surl.Host
	h := sumdbProxy(surl, nil)
	h = http.StripPrefix(pathPrefix, h)

	targetURL := "/sumdb/" + surl.Host + "/latest"
	req := httptest.NewRequest("GET", targetURL, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)

	require.Equal(t, expectedURL, givenURL)

}

var noSumTestCases = []struct {
	name     string
	patterns []string
	given    string
	status   int
}{
	{
		"no match",
		[]string{"github.com/private/repo"},
		"github.com/public/repo@v0.0.1",
		http.StatusOK,
	},
	{
		"exact match",
		[]string{"github.com/private/repo@v0.0.1"},
		"github.com/private/repo@v0.0.1",
		http.StatusForbidden,
	},
	{
		"star match",
		[]string{"github.com/private/*"},
		"github.com/private/repo@v0.0.1",
		http.StatusForbidden,
	},
	{
		"multi slash star",
		[]string{"github.com/private/*"},
		"github.com/private/repo/sub@v0.0.1",
		http.StatusForbidden,
	},
	{
		"multi star",
		[]string{"github.com/*/*"},
		"github.com/private/repo@v0.0.1",
		http.StatusForbidden,
	},
	{
		"multi star ok",
		[]string{"github.com/private/*/*"},
		"github.com/private/repo@v0.0.1",
		http.StatusOK,
	},
	{
		"multi star forbidden",
		[]string{"github.com/private/*/*"},
		"github.com/private/repo/sub@v0.0.1",
		http.StatusForbidden,
	},
	{
		"any version",
		[]string{"github.com/private/repo*"},
		"github.com/private/repo@v0.0.1",
		http.StatusForbidden,
	},
}

func TestNoSumPatterns(t *testing.T) {
	for _, tc := range noSumTestCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			skipHandler := noSumWrapper(http.HandlerFunc(emptyHandler), tc.patterns)
			req := httptest.NewRequest("GET", "/lookup/"+tc.given, nil)
			skipHandler.ServeHTTP(w, req)
			require.Equal(t, w.Code, tc.status)

		})
	}
}

func emptyHandler(w http.ResponseWriter, r *http.Request) {}
