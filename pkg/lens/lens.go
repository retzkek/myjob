package lens

import (
	"fmt"
    "os"

	"github.com/machinebox/graphql"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Lens is a client interface to the Lanscape Lens GraphQL API.
type Lens struct {
	URL    string
	client *graphql.Client
}

// NewLensClient initializes a Lens client.
func NewLensClient(url string) *Lens {
	l := &Lens{
		URL:    url,
		client: graphql.NewClient(url),
	}
	return l
}

var defaultClient *Lens

func init() {
	defaultClient = NewLensClient(os.Getenv("LENS_URL"))
}

func spanError(span opentracing.Span, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	ext.LogError(span, err)
	return err
}
