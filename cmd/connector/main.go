package main

import (
	arxiv "github.com/raulb/conduit-connector-arxiv"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

func main() {
	sdk.Serve(arxiv.Connector)
}
