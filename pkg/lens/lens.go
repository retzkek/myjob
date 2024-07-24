package lens

import (
	"fmt"
	"os"

	"github.com/machinebox/graphql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Lens is a client interface to the Lanscape Lens GraphQL API.
type Lens struct {
	URL    string
	client *graphql.Client
	tracer trace.Tracer
}

// NewLensClient initializes a Lens client.
func NewLensClient(url string) *Lens {
	l := &Lens{
		URL:    url,
		client: graphql.NewClient(url),
		tracer: otel.Tracer("lensClient"),
	}
	return l
}

var defaultClient *Lens

func init() {
	defaultClient = NewLensClient(os.Getenv("LENS_URL"))
}

func spanError(span trace.Span, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	span.SetStatus(codes.Error, err.Error())
	return err
}
