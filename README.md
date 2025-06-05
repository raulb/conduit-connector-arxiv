# Conduit Connector for <!-- readmegen:name -->Arxiv<!-- /readmegen:name -->

[Conduit](https://conduit.io) connector for <!-- readmegen:name -->Arxiv<!-- /readmegen:name -->.

<!-- readmegen:description -->
The arXiv connector fetches academic papers from the arXiv API based on configurable search queries.

It supports filtering by categories, date ranges, and search terms. The connector can fetch paper metadata
and optionally download PDF files. It's designed to work with arXiv's REST API and handles pagination
and rate limiting automatically.

Features:
- Search by keywords, categories, and authors
- Configurable polling intervals
- PDF download support
- Metadata extraction (title, abstract, authors, categories)
- Position tracking for incremental updates<!-- /readmegen:description -->

## Source

A source connector pulls data from an external resource and pushes it to
downstream resources via Conduit.

### Configuration

<!-- readmegen:source.parameters.yaml -->
```yaml
version: 2.2
pipelines:
  - id: example
    type: source
    status: running
    connectors:
      - id: example
        plugin: "arxiv"
        settings:
          # SearchQuery is the arXiv search query (e.g., "ti:\"AI\" AND
          # cat:cs.AI")
          # Type: string
          # Required: yes
          search_query: ""
          # ArxivAPIURL is the base URL for the arXiv API
          # Type: string
          # Required: no
          arxiv_api_url: "https://export.arxiv.org/api/query"
          # FilterLast24Hours only fetches papers from the last 24 hours
          # Type: bool
          # Required: no
          filter_last_24_hours: "false"
          # IncludePDF determines if PDF URLs should be included in the output
          # Type: bool
          # Required: no
          include_pdf: "true"
          # MaxResults is the maximum number of results to fetch per request
          # (default: 100)
          # Type: int
          # Required: no
          max_results: "100"
          # PollingPeriod is how often to poll for new papers
          # Type: duration
          # Required: no
          polling_period: "1h"
          # SortBy determines how to sort results (submittedDate,
          # lastUpdatedDate, relevance)
          # Type: string
          # Required: no
          sort_by: "submittedDate"
          # SortOrder determines sort order (ascending, descending)
          # Type: string
          # Required: no
          sort_order: "descending"
          # Maximum delay before an incomplete batch is read from the source.
          # Type: duration
          # Required: no
          sdk.batch.delay: "0"
          # Maximum size of batch before it gets read from the source.
          # Type: int
          # Required: no
          sdk.batch.size: "0"
          # Specifies whether to use a schema context name. If set to false, no
          # schema context name will be used, and schemas will be saved with the
          # subject name specified in the connector (not safe because of name
          # conflicts).
          # Type: bool
          # Required: no
          sdk.schema.context.enabled: "true"
          # Schema context name to be used. Used as a prefix for all schema
          # subject names. If empty, defaults to the connector ID.
          # Type: string
          # Required: no
          sdk.schema.context.name: ""
          # Whether to extract and encode the record key with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.key.enabled: "true"
          # The subject of the key schema. If the record metadata contains the
          # field "opencdc.collection" it is prepended to the subject name and
          # separated with a dot.
          # Type: string
          # Required: no
          sdk.schema.extract.key.subject: "key"
          # Whether to extract and encode the record payload with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.payload.enabled: "true"
          # The subject of the payload schema. If the record metadata contains
          # the field "opencdc.collection" it is prepended to the subject name
          # and separated with a dot.
          # Type: string
          # Required: no
          sdk.schema.extract.payload.subject: "payload"
          # The type of the payload schema.
          # Type: string
          # Required: no
          sdk.schema.extract.type: "avro"
```
<!-- /readmegen:source.parameters.yaml -->

## Destination

A destination connector pushes data from upstream resources to an external
resource via Conduit.

### Configuration

<!-- readmegen:destination.parameters.yaml -->
```yaml
version: 2.2
pipelines:
  - id: example
    type: destination
    status: running
    connectors:
      - id: example
        plugin: "arxiv"
        settings:
```
<!-- /readmegen:destination.parameters.yaml -->

## Development

- To install the required tools, run `make install-tools`.
- To generate code (mocks, re-generate `connector.yaml`, update the README,
  etc.), run `make generate`.

## How to build?

Run `make build` to build the connector.

## Testing

Run `make test` to run all the unit tests. Run `make test-integration` to run
the integration tests.

The Docker compose file at `test/docker-compose.yml` can be used to run the
required resource locally.

## How to release?

The release is done in two steps:

- Bump the version in [connector.yaml](/connector.yaml). This can be done
  with [bump_version.sh](/scripts/bump_version.sh) script, e.g.
  `scripts/bump_version.sh 2.3.4` (`2.3.4` is the new version and needs to be a
  valid semantic version). This will also automatically create a PR for the
  change.
- Tag the connector, which will kick off a release. This can be done
  with [tag.sh](/scripts/tag.sh).

## Known Issues & Limitations

- Known issue A
- Limitation A

## Planned work

- [ ] Item A
- [ ] Item B
