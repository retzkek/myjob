package lens

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/machinebox/graphql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

type Job struct {
	ID         string
	Owner      string
	Group      string
	Subject    string
	SubmitTime time.Time
	Done       bool
}

var (
	submissionQuery = `
query {
  job:submission(id:"%s"){
    id owner group
    subject: classAd(name: "AuthTokenSubject")
    submitTime
    done
  }
}
`

	jobQuery = `
query {
  job(id:"%s"){
    id owner group
    subject: classAd(name: "AuthTokenSubject")
    submitTime
    done
  }
}
`
	jobOrSubmissionIDRegexp = regexp.MustCompile("(\\w+)(\\.\\d+)?@([\\w\\.]+)")
)

func GetJobInfo(ctx context.Context, jobid string) (*Job, error) {
	return defaultClient.GetJobInfo(ctx, jobid)
}

// GetJobInfo looks up the information for the job/submission.
func (l *Lens) GetJobInfo(ctx context.Context, jobid string) (*Job, error) {
	ctx, span := l.tracer.Start(ctx, "GetJobInfo")
	defer span.End()
	span.SetAttributes(attribute.String("job.id", jobid))

	if l.client == nil {
		return nil, spanError(span, "Lens client was not initialized")
	}

	// try to determine if this is a job or a submission
	m := jobOrSubmissionIDRegexp.FindStringSubmatch(jobid)
	if len(m) < 4 || len(m[1]) == 0 || len(m[3]) == 0 {
		return nil, spanError(span, "\"%s\" does not appear to be a job or submission id", jobid)
	}
	var q string
	if len(m[2]) == 0 {
		// no process id, so it must be a submission
		q = fmt.Sprintf(submissionQuery, jobid)
	} else {
		// ... otherwise it's a job
		q = fmt.Sprintf(jobQuery, jobid)
	}

	req := graphql.NewRequest(q)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	var resp struct {
		Job *Job
	}
	if err := l.client.Run(ctx, req, &resp); err != nil {
		return nil, spanError(span, err.Error())
	}
	if resp.Job == nil {
		return nil, spanError(span, "job info missing from response")
	}
	return resp.Job, nil

}
