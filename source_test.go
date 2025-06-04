package arxiv_test

import (
	"context"
	"testing"

	arxiv "github.com/raulb/conduit-connector-arxiv"
	"github.com/matryer/is"
)

func TestTeardownSource_NoOpen(t *testing.T) {
	is := is.New(t)
	con := arxiv.NewSource()
	err := con.Teardown(context.Background())
	is.NoErr(err)
}
