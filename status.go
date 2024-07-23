package main

import (
	"fmt"
	"context"

	"github.com/clok/kemba"
	"github.com/urfave/cli/v2"

	"github.com/retzkek/myjob/pkg/lens"
)

var StatusFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "jobid",
		Aliases:  []string{"J", "job"},
		Usage:    "job/submission ID",
	},
}

func Status(ctx *cli.Context) error {
	switch {
	case ctx.IsSet("jobid"):
		return JobStatus(ctx,ctx.String("jobid"))
	}
	return nil
}
	
func JobStatus(ctx *cli.Context, jobid string) error {
	k := kemba.New("myjob:status:job")

	k.Printf("getting info for job/submission %s from lens",jobid)
	j, err := lens.GetJobInfo(context.Background(), jobid)
	if err != nil {
		return err
	}
	k.Println(j)

	done:="not done"
	if j.Done {
		done = "done"
	}
	fmt.Printf("Subission %s submitted by %s at %s is %s.\n",jobid,j.Owner,j.SubmitTime.String(),done)
	return nil
}

