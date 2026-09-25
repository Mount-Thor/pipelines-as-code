package gitea

import (
	"net/http"
	"testing"

	forgejo "codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v3"
	"gotest.tools/v3/assert"
)

func newPagedResponse(header http.Header) *forgejo.Response {
	return &forgejo.Response{Response: &http.Response{Header: header}}
}

func TestShouldGetNextPage(t *testing.T) {
	tests := []struct {
		name        string
		resp        *forgejo.Response
		currentPage int
		wantNext    bool
		wantPage    int
	}{
		{
			name:        "x-pagecount 0 stops (empty PR / deleted head branch)",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"0"}}),
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
		{
			name:        "x-pagecount 0 stops even deep into pagination",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"0"}}),
			currentPage: 4900000,
			wantNext:    false,
			wantPage:    4900000,
		},
		{
			name:        "x-pagecount missing stops",
			resp:        newPagedResponse(http.Header{}),
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
		{
			name:        "x-pagecount present but empty stops",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {}}),
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
		{
			name:        "x-pagecount not a number stops",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"many"}}),
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
		{
			name:        "x-pagecount greater than current page fetches the next page",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"3"}}),
			currentPage: 1,
			wantNext:    true,
			wantPage:    2,
		},
		{
			name:        "x-pagecount equal to current page stops on the last page",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"3"}}),
			currentPage: 3,
			wantNext:    false,
			wantPage:    3,
		},
		{
			name:        "x-pagecount 1 with a single page stops",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"1"}}),
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
		{
			name:        "current page already past x-pagecount stops",
			resp:        newPagedResponse(http.Header{"X-Pagecount": {"2"}}),
			currentPage: 5,
			wantNext:    false,
			wantPage:    5,
		},
		{
			name:        "lower-case header key is canonicalised",
			resp:        &forgejo.Response{Response: &http.Response{Header: http.Header{http.CanonicalHeaderKey("x-pagecount"): {"2"}}}},
			currentPage: 1,
			wantNext:    true,
			wantPage:    2,
		},
		{
			name:        "nil response stops",
			resp:        nil,
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
		{
			name:        "nil embedded http response stops",
			resp:        &forgejo.Response{},
			currentPage: 1,
			wantNext:    false,
			wantPage:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNext, gotPage := ShouldGetNextPage(tt.resp, tt.currentPage)
			assert.Equal(t, gotNext, tt.wantNext, "shouldGetNextPage")
			assert.Equal(t, gotPage, tt.wantPage, "next page")
		})
	}
}

// TestShouldGetNextPageTerminates walks the same loop shape as
// fetchChangedFiles and asserts it terminates for every page count Forgejo
// can return, including 0, with a bounded number of iterations.
func TestShouldGetNextPageTerminates(t *testing.T) {
	for _, pagecount := range []string{"0", "1", "2", "7"} {
		t.Run("x-pagecount="+pagecount, func(t *testing.T) {
			resp := newPagedResponse(http.Header{"X-Pagecount": {pagecount}})
			page := 1
			iterations := 0
			for {
				iterations++
				if iterations > 100 {
					t.Fatalf("pagination did not terminate for x-pagecount=%s (reached page %d)", pagecount, page)
				}
				next, nextPage := ShouldGetNextPage(resp, page)
				if !next {
					break
				}
				page = nextPage
			}
			want := 1
			if pagecount != "0" {
				want = int(pagecount[0] - '0')
			}
			assert.Equal(t, iterations, want, "requests issued")
		})
	}
}
