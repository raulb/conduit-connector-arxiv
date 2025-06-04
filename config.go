package arxiv

import (
	"fmt"
	"net/url"
)

// Config contains shared config parameters, common to the source and
// destination. If you don't need shared parameters you can entirely remove this
// file.
type Config struct {
	// ArxivAPIURL is the base URL for the arXiv API
	ArxivAPIURL string `json:"arxiv_api_url" default:"https://export.arxiv.org/api/query"`
}

// buildArxivURL constructs the arXiv API URL with the given parameters
func (c Config) buildArxivURL(searchQuery, sortBy, sortOrder string, start, maxResults int) (string, error) {
	baseURL, err := url.Parse(c.ArxivAPIURL)
	if err != nil {
		return "", fmt.Errorf("invalid arXiv API URL: %w", err)
	}

	params := url.Values{}
	params.Set("search_query", searchQuery)
	params.Set("sortBy", sortBy)
	params.Set("sortOrder", sortOrder)
	params.Set("start", fmt.Sprintf("%d", start))
	params.Set("max_results", fmt.Sprintf("%d", maxResults))

	baseURL.RawQuery = params.Encode()
	return baseURL.String(), nil
}
