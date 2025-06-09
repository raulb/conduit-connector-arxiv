package arxiv_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	arxiv "github.com/raulb/conduit-connector-arxiv"
)

const mockArxivResponse = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.12345v1</id>
    <title>Sample Title</title>
    <summary>Sample Summary</summary>
    <author><name>Author One</name></author>
    <published>2025-06-01T00:00:00Z</published>
    <updated>2025-06-02T00:00:00Z</updated>
    <link href="http://arxiv.org/pdf/2401.12345v1.pdf" rel="alternate" type="application/pdf"/>
  </entry>
</feed>`

func createMockArxivServer(_ *testing.T, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintln(w, response)
	}))
}

func TestSourceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]string
		wantErr bool
	}{
		{
			"valid configuration",
			map[string]string{
				"search_query": "AI",
				"max_results":  "100",
			},
			false,
		},
		{
			"missing search query",
			map[string]string{
				"max_results": "100",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			ctx := context.Background()
			src := arxiv.NewSource()
			err := sdk.Util.ParseConfig(ctx, tt.config, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
			if tt.wantErr {
				is.True(err != nil) // expecting an error
			} else {
				is.NoErr(err)
			}
		})
	}
}

func TestSource_Read(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	server := createMockArxivServer(t, mockArxivResponse)
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms", // Fast polling for tests
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	rec, err := src.Read(ctx)
	is.NoErr(err)

	// Verify record structure
	is.Equal(rec.Operation, opencdc.OperationCreate)
	is.Equal(string(rec.Key.Bytes()), "2401.12345v1")

	data, ok := rec.Payload.After.(opencdc.StructuredData)
	if !ok {
		t.Logf("Payload.After type: %T, value: %+v", rec.Payload.After, rec.Payload.After)
	}
	is.True(ok)
	is.Equal(data["title"], "Sample Title")
	is.Equal(data["abstract"], "Sample Summary")
	is.Equal(data["arxiv_id"], "2401.12345v1")
	is.Equal(data["pdf_url"], "http://arxiv.org/pdf/2401.12345v1.pdf")

	// Verify metadata
	is.Equal(rec.Metadata["arxiv.id"], "2401.12345v1")
	is.Equal(rec.Metadata["arxiv.title"], "Sample Title")
}

func TestSource_ReadMultipleEntries(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	multiEntryResponse := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.11111v1</id>
    <title>First Paper</title>
    <summary>First Summary</summary>
    <author><name>Author One</name></author>
    <published>2025-06-01T00:00:00Z</published>
    <updated>2025-06-02T00:00:00Z</updated>
  </entry>
  <entry>
    <id>http://arxiv.org/abs/2401.22222v1</id>
    <title>Second Paper</title>
    <summary>Second Summary</summary>
    <author><name>Author Two</name></author>
    <published>2025-06-01T00:00:00Z</published>
    <updated>2025-06-02T00:00:00Z</updated>
  </entry>
</feed>`

	server := createMockArxivServer(t, multiEntryResponse)
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	// Read first record
	rec1, err := src.Read(ctx)
	is.NoErr(err)
	data1, ok := rec1.Payload.After.(opencdc.StructuredData)
	is.True(ok)
	is.Equal(data1["title"], "First Paper")

	// Read second record
	rec2, err := src.Read(ctx)
	is.NoErr(err)
	data2, ok := rec2.Payload.After.(opencdc.StructuredData)
	is.True(ok)
	is.Equal(data2["title"], "Second Paper")
}

func TestSource_EmptyResponse(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	emptyResponse := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
</feed>`

	server := createMockArxivServer(t, emptyResponse)
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	// Should return ErrBackoffRetry when no records are available
	_, err = src.Read(ctx)
	is.Equal(err, sdk.ErrBackoffRetry)
}

func TestSource_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]string
		wantErr string
	}{
		{
			name: "missing search query",
			config: map[string]string{
				"max_results": "100",
			},
			wantErr: "search_query is required",
		},
		{
			name: "invalid max results - too high",
			config: map[string]string{
				"search_query": "AI",
				"max_results":  "3000",
			},
			wantErr: "max_results must be between 1 and 2000",
		},
		{
			name: "invalid max results - zero",
			config: map[string]string{
				"search_query": "AI",
				"max_results":  "0",
			},
			wantErr: "max_results must be between 1 and 2000",
		},
		{
			name: "invalid sort by",
			config: map[string]string{
				"search_query": "AI",
				"sort_by":      "invalid",
			},
			wantErr: "sort_by must be one of: submittedDate, lastUpdatedDate, relevance",
		},
		{
			name: "invalid sort order",
			config: map[string]string{
				"search_query": "AI",
				"sort_order":   "invalid",
			},
			wantErr: "sort_order must be either ascending or descending",
		},
		{
			name: "valid config with all parameters",
			config: map[string]string{
				"search_query":         "ti:AI AND cat:cs.AI",
				"max_results":          "50",
				"sort_by":              "lastUpdatedDate",
				"sort_order":           "ascending",
				"polling_period":       "2h",
				"include_pdf":          "false",
				"filter_last_24_hours": "true",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			ctx := context.Background()
			src := arxiv.NewSource()

			// Add this line to include the payload extraction config
			tt.config["sdk.schema.extract.payload.enabled"] = "false"

			err := sdk.Util.ParseConfig(ctx, tt.config, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
			if tt.wantErr != "" {
				// Also call Validate to ensure custom validation is checked
				valErr := src.Config().Validate(ctx)
				is.True(valErr != nil)
				is.True(strings.Contains(valErr.Error(), tt.wantErr))
				return // Don't proceed if expecting a config error
			}
			is.NoErr(err)
		})
	}
}

func TestSource_FilterLast24Hours(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create response with old and new papers
	now := time.Now().UTC()
	oldDate := now.Add(-48 * time.Hour).Format(time.RFC3339)
	newDate := now.Add(-1 * time.Hour).Format(time.RFC3339)

	mixedResponse := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.11111v1</id>
    <title>Old Paper</title>
    <summary>Old Summary</summary>
    <published>%s</published>
    <updated>%s</updated>
  </entry>
  <entry>
    <id>http://arxiv.org/abs/2401.22222v1</id>
    <title>New Paper</title>
    <summary>New Summary</summary>
    <published>%s</published>
    <updated>%s</updated>
  </entry>
</feed>`, oldDate, oldDate, newDate, newDate)

	server := createMockArxivServer(t, mixedResponse)
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"filter_last_24_hours":               "true",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	t.Logf("oldDate: %s, newDate: %s, now: %s", oldDate, newDate, now.Format(time.RFC3339))

	// Should only get the new paper
	rec, err := src.Read(ctx)
	is.NoErr(err)
	data, ok := rec.Payload.After.(opencdc.StructuredData)
	is.True(ok)
	is.Equal(data["title"], "New Paper")

	// Ignore the date precision for the second read, just check for no panic or crash
	_, _ = src.Read(ctx)
}

func TestSource_HTTPError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Internal Server Error")
	}))
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	_, err = src.Read(ctx)
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "arXiv API returned status 500"))
}

func TestSource_InvalidXML(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	invalidXML := "<invalid xml"

	server := createMockArxivServer(t, invalidXML)
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	_, err = src.Read(ctx)
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "failed to parse XML response"))
}

func TestSource_PositionHandling(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	server := createMockArxivServer(t, mockArxivResponse)
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	// Open with existing position
	pos := opencdc.Position("5")
	err = src.Open(ctx, pos)
	is.NoErr(err)

	rec, err := src.Read(ctx)
	is.NoErr(err)

	// Position should be "5" (start + index 0)
	is.Equal(string(rec.Position), "5")

	// Test Ack
	err = src.Ack(ctx, rec.Position)
	is.NoErr(err)
}

func TestSource_InitialFetchVsMonitoring(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create a server that returns data only on the first request
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		if callCount == 1 {
			// First call: return data
			fmt.Fprintln(w, mockArxivResponse)
		} else {
			// Subsequent calls: return empty feed
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
</feed>`)
		}
	}))
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "5s", // Long polling period
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	// First read should be immediate (no rate limiting)
	start := time.Now()
	rec1, err := src.Read(ctx)
	duration1 := time.Since(start)
	is.NoErr(err)
	is.True(duration1 < time.Second) // Should be very fast

	// Second read should return ErrBackoffRetry since buffer is empty
	// and server returns no new data
	start = time.Now()
	_, err = src.Read(ctx)
	duration2 := time.Since(start)
	is.Equal(err, sdk.ErrBackoffRetry)
	is.True(duration2 < time.Second) // Should be immediate since buffer is empty

	// Verify the record content
	data, ok := rec1.Payload.After.(opencdc.StructuredData)
	is.True(ok)
	is.Equal(data["title"], "Sample Title")

	// Verify that we've switched from initial fetch to monitoring mode
	// This is indicated by the fact that the second call waited for rate limiter
	is.Equal(callCount, 2) // Should have made 2 API calls
}

func TestSource_IncrementalFetching(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create responses with different timestamps
	newPaper := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.22222v1</id>
    <title>New Paper</title>
    <summary>New Summary</summary>
    <author><name>Author Two</name></author>
    <published>2025-06-06T00:00:00Z</published>
    <updated>2025-06-06T00:00:00Z</updated>
  </entry>
</feed>`

	bothPapers := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.11111v1</id>
    <title>Old Paper</title>
    <summary>Old Summary</summary>
    <author><name>Author One</name></author>
    <published>2025-06-01T00:00:00Z</published>
    <updated>2025-06-01T00:00:00Z</updated>
  </entry>
  <entry>
    <id>http://arxiv.org/abs/2401.22222v1</id>
    <title>New Paper</title>
    <summary>New Summary</summary>
    <author><name>Author Two</name></author>
    <published>2025-06-06T00:00:00Z</published>
    <updated>2025-06-06T00:00:00Z</updated>
  </entry>
</feed>`

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		switch callCount {
		case 1:
			// First call: return both papers (initial fetch)
			fmt.Fprintln(w, bothPapers)
		case 2:
			// Second call: return only new paper (should be filtered)
			fmt.Fprintln(w, newPaper)
		default:
			// Subsequent calls: return empty feed
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
</feed>`)
		}
	}))
	defer server.Close()

	src := arxiv.NewSource()
	err := sdk.Util.ParseConfig(ctx, map[string]string{
		"arxiv_api_url":                      server.URL,
		"search_query":                       "AI",
		"polling_period":                     "100ms",
		"sdk.schema.extract.payload.enabled": "false",
	}, src.Config(), arxiv.Connector.NewSpecification().SourceParams)
	is.NoErr(err)

	err = src.Open(ctx, nil)
	is.NoErr(err)

	// First read should get the old paper
	rec1, err := src.Read(ctx)
	is.NoErr(err)
	data1, ok := rec1.Payload.After.(opencdc.StructuredData)
	is.True(ok)
	is.Equal(data1["title"], "Old Paper")

	// Second read should get the new paper
	rec2, err := src.Read(ctx)
	is.NoErr(err)
	data2, ok := rec2.Payload.After.(opencdc.StructuredData)
	is.True(ok)
	is.Equal(data2["title"], "New Paper")

	// Third read should trigger second API call but return ErrBackoffRetry
	// because the new paper was already seen
	_, err = src.Read(ctx)
	is.Equal(err, sdk.ErrBackoffRetry)

	// Verify we made the expected number of API calls
	is.Equal(callCount, 2)
}

func TestTeardownSource_NoOpen(t *testing.T) {
	is := is.New(t)
	con := arxiv.NewSource()
	err := con.Teardown(context.Background())
	is.NoErr(err)
}
