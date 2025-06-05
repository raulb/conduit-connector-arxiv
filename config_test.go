package arxiv_test

import (
	"testing"

	"github.com/matryer/is"
	arxiv "github.com/raulb/conduit-connector-arxiv"
)

func TestConfig_BuildArxivURL(t *testing.T) {
	tests := []struct {
		name        string
		config      arxiv.Config
		searchQuery string
		sortBy      string
		sortOrder   string
		start       int
		maxResults  int
		expected    string
		wantErr     bool
	}{
		{
			name: "basic query",
			config: arxiv.Config{
				ArxivAPIURL: "https://export.arxiv.org/api/query",
			},
			searchQuery: "AI",
			sortBy:      "submittedDate",
			sortOrder:   "descending",
			start:       0,
			maxResults:  10,
			expected:    "https://export.arxiv.org/api/query?max_results=10&search_query=AI&sortBy=submittedDate&sortOrder=descending&start=0",
			wantErr:     false,
		},
		{
			name: "complex query with categories",
			config: arxiv.Config{
				ArxivAPIURL: "https://export.arxiv.org/api/query",
			},
			searchQuery: "ti:artificial intelligence AND cat:cs.AI",
			sortBy:      "lastUpdatedDate",
			sortOrder:   "ascending",
			start:       100,
			maxResults:  50,
			expected:    "https://export.arxiv.org/api/query?max_results=50&search_query=ti%3Aartificial+intelligence+AND+cat%3Acs.AI&sortBy=lastUpdatedDate&sortOrder=ascending&start=100",
			wantErr:     false,
		},
		{
			name: "invalid base URL",
			config: arxiv.Config{
				ArxivAPIURL: "://invalid-url",
			},
			searchQuery: "AI",
			sortBy:      "submittedDate",
			sortOrder:   "descending",
			start:       0,
			maxResults:  10,
			expected:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			result, err := tt.config.BuildArxivURL(
				tt.searchQuery,
				tt.sortBy,
				tt.sortOrder,
				tt.start,
				tt.maxResults,
			)

			if tt.wantErr {
				is.True(err != nil)
			} else {
				is.NoErr(err)
				is.Equal(result, tt.expected)
			}
		})
	}
}

