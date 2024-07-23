package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"

	"github.com/retzkek/myjob/pkg/lens"
)

var (
	address = flag.String("a", "localhost:8888", "Address and port to listen on")
)

func main() {
	flag.Parse()

	http.HandleFunc("/status/{jobid}", func(w http.ResponseWriter, r *http.Request) {
		if err := JobStatus(r.Context(), r.PathValue("jobid"), w); err != nil {
			http.Error(w, err.Error(), 503)
		}
	})

	fmt.Println("Listening on", *address)
	http.ListenAndServe(*address, nil)
}

func JobStatus(ctx context.Context, jobid string, w io.Writer) error {
	j, err := lens.GetJobInfo(ctx, jobid)
	if err != nil {
		return err
	}

	done := "not done"
	if j.Done {
		done = "done"
	}
	fmt.Fprintf(w, "Subission %s submitted by %s at %s is %s.\n", jobid, j.Owner, j.SubmitTime.String(), done)
	return nil
}
