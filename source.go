package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"golang.org/x/time/rate"
)

// ArxivEntry represents a single paper entry from arXiv
type ArxivEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Summary   string     `xml:"summary"`
	Authors   []Author   `xml:"author"`
	Published time.Time  `xml:"published"`
	Updated   time.Time  `xml:"updated"`
	Links     []Link     `xml:"link"`
	Category  []Category `xml:"category"`
}

func (e *ArxivEntry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type Alias ArxivEntry
	tmp := &struct {
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
		*Alias
	}{Alias: (*Alias)(e)}
	if err := d.DecodeElement(tmp, &start); err != nil {
		return err
	}
	var err error
	e.Published, err = time.Parse(time.RFC3339, tmp.Published)
	if err != nil {
		// Try parsing without timezone (arXiv sometimes omits 'Z')
		e.Published, err = time.Parse("2006-01-02T15:04:05", tmp.Published)
		if err != nil {
			return fmt.Errorf("failed to parse published date: %w", err)
		}
	}
	e.Updated, err = time.Parse(time.RFC3339, tmp.Updated)
	if err != nil {
		// Try parsing without timezone
		e.Updated, err = time.Parse("2006-01-02T15:04:05", tmp.Updated)
		if err != nil {
			return fmt.Errorf("failed to parse updated date: %w", err)
		}
	}
	return nil
}

type Author struct {
	Name string `xml:"name"`
}

type Link struct {
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
	Rel  string `xml:"rel,attr"`
}

type Category struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

// ArxivFeed represents the XML response from arXiv API
type ArxivFeed struct {
	XMLName xml.Name      `xml:"feed"`
	Entries []*ArxivEntry `xml:"entry"`
	Title   string        `xml:"title"`
}

type Source struct {
	sdk.UnimplementedSource

	config  SourceConfig
	client  *http.Client
	limiter *rate.Limiter

	buffer       []opencdc.Record
	lastPosition opencdc.Position
	offset       int
}

type SourceConfig struct {
	sdk.DefaultSourceMiddleware
	// Config includes parameters that are the same in the source and destination.
	Config

	// SearchQuery is the arXiv search query (e.g., "ti:\"AI\" AND cat:cs.AI")
	SearchQuery string `json:"search_query" validate:"required"`

	// MaxResults is the maximum number of results to fetch per request (default: 100)
	MaxResults int `json:"max_results" default:"100"`

	// SortBy determines how to sort results (submittedDate, lastUpdatedDate, relevance)
	SortBy string `json:"sort_by" default:"submittedDate"`

	// SortOrder determines sort order (ascending, descending)
	SortOrder string `json:"sort_order" default:"descending"`

	// PollingPeriod is how often to poll for new papers
	PollingPeriod time.Duration `json:"polling_period" default:"1h"`

	// IncludePDF determines if PDF URLs should be included in the output
	IncludePDF bool `json:"include_pdf" default:"true"`

	// FilterLast24Hours only fetches papers from the last 24 hours
	FilterLast24Hours bool `json:"filter_last_24_hours" default:"false"`
}

func (s *SourceConfig) Validate(ctx context.Context) error {
	// Validate the configuration
	if err := s.DefaultSourceMiddleware.Validate(ctx); err != nil {
		return err
	}

	if s.SearchQuery == "" {
		return fmt.Errorf("search_query is required")
	}

	if s.MaxResults <= 0 || s.MaxResults > 2000 {
		return fmt.Errorf("max_results must be between 1 and 2000")
	}

	validSortBy := map[string]bool{
		"submittedDate":   true,
		"lastUpdatedDate": true,
		"relevance":       true,
	}
	if !validSortBy[s.SortBy] {
		return fmt.Errorf("sort_by must be one of: submittedDate, lastUpdatedDate, relevance")
	}

	validSortOrder := map[string]bool{
		"ascending":  true,
		"descending": true,
	}
	if !validSortOrder[s.SortOrder] {
		return fmt.Errorf("sort_order must be either ascending or descending")
	}

	return nil
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{})
}

func (s *Source) Config() sdk.SourceConfig {
	return &s.config
}

func (s *Source) Open(ctx context.Context, pos opencdc.Position) error {
	sdk.Logger(ctx).Info().Msg("opening arXiv source")

	s.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Set up rate limiter based on polling period
	s.limiter = rate.NewLimiter(rate.Every(s.config.PollingPeriod), 1)

	// Parse position to get offset
	if len(pos) > 0 {
		if offset, err := strconv.Atoi(string(pos)); err == nil {
			s.offset = offset
		}
	}

	s.lastPosition = pos
	return nil
}

func (s *Source) Read(ctx context.Context) (opencdc.Record, error) {
	if len(s.buffer) == 0 {
		// Wait for rate limiter
		err := s.limiter.Wait(ctx)
		if err != nil {
			return opencdc.Record{}, err
		}

		// Fill buffer with new records
		err = s.fillBuffer(ctx)
		if err != nil {
			return opencdc.Record{}, err
		}
	}

	if len(s.buffer) == 0 {
		return opencdc.Record{}, sdk.ErrBackoffRetry
	}

	// Return the first record from buffer
	rec := s.buffer[0]
	s.buffer = s.buffer[1:]
	s.lastPosition = rec.Position

	return rec, nil
}

func (s *Source) fillBuffer(ctx context.Context) error {
	sdk.Logger(ctx).Debug().Msg("filling buffer with arXiv entries")

	// Build arXiv API URL
	apiURL, err := s.config.BuildArxivURL(
		s.config.SearchQuery,
		s.config.SortBy,
		s.config.SortOrder,
		s.offset,
		s.config.MaxResults,
	)
	if err != nil {
		return fmt.Errorf("failed to build arXiv URL: %w", err)
	}

	// Make request to arXiv API
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Conduit-ArXiv-Connector/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch from arXiv: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("arXiv API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var feed ArxivFeed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return fmt.Errorf("failed to parse XML response: %w", err)
	}

	// Convert entries to OpenCDC records
	for i, entry := range feed.Entries {
		// Apply 24-hour filter if enabled
		if s.config.FilterLast24Hours {
			twentyFourHoursAgo := time.Now().UTC().Add(-24 * time.Hour)
			if entry.Published.Before(twentyFourHoursAgo) {
				continue
			}
		}

		rec, err := s.entryToRecord(*entry, s.offset+i)
		if err != nil {
			return fmt.Errorf("failed to convert entry to record: %w", err)
		}
		s.buffer = append(s.buffer, rec)
	}

	// Update offset for next request
	s.offset += len(feed.Entries)

	return nil
}

func (s *Source) entryToRecord(entry ArxivEntry, position int) (opencdc.Record, error) {
	// Extract arXiv ID from the entry ID URL
	arxivID := extractArxivID(entry.ID)

	// Find PDF link if requested
	var pdfURL string
	if s.config.IncludePDF {
		for _, link := range entry.Links {
			if link.Type == "application/pdf" || (link.Rel == "alternate" && link.Type == "") {
				pdfURL = link.Href
				break
			}
		}
	}

	// Extract author names
	authors := make([]string, len(entry.Authors))
	for i, author := range entry.Authors {
		authors[i] = author.Name
	}

	// Extract categories
	categories := make([]string, len(entry.Category))
	for i, cat := range entry.Category {
		categories[i] = cat.Term
	}

	// Create structured data
	data := map[string]interface{}{
		"arxiv_id":   arxivID,
		"title":      entry.Title,
		"abstract":   entry.Summary,
		"authors":    authors,
		"published":  entry.Published.Format(time.RFC3339),
		"updated":    entry.Updated.Format(time.RFC3339),
		"categories": categories,
		"entry_url":  entry.ID,
	}

	if pdfURL != "" {
		data["pdf_url"] = pdfURL
	}

	// Create metadata
	meta := opencdc.Metadata{}
	meta.SetReadAt(time.Now())
	meta["arxiv.id"] = arxivID
	meta["arxiv.title"] = entry.Title
	meta["arxiv.published"] = entry.Published.Format(time.RFC3339)

	return opencdc.Record{
		Operation: opencdc.OperationCreate,
		Position:  opencdc.Position(strconv.Itoa(position)),
		Key:       opencdc.RawData(arxivID),
		Payload: opencdc.Change{
			After: opencdc.StructuredData(data),
		},
		Metadata: meta,
	}, nil
}

// extractArxivID extracts the arXiv ID from the entry ID URL
func extractArxivID(entryID string) string {
	// Entry ID format: http://arxiv.org/abs/1234.5678v1
	parts := strings.Split(entryID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return entryID
}

func (s *Source) Ack(ctx context.Context, position opencdc.Position) error {
	sdk.Logger(ctx).Debug().Str("position", string(position)).Msg("got ack")
	return nil
}

func (s *Source) Teardown(ctx context.Context) error {
	sdk.Logger(ctx).Info().Msg("tearing down arXiv source")
	if s.client != nil {
		// Close any connections if needed
	}
	return nil
}
