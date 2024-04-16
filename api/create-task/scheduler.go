package p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	scheduler "cloud.google.com/go/scheduler/apiv1"
	schedulerpb "google.golang.org/genproto/googleapis/cloud/scheduler/v1"

	"google.golang.org/protobuf/types/known/durationpb"
)

func createScheduler(ctx context.Context, projectID string, t *task, region, uri string) error {
	c, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return fmt.Errorf("scheduler.NewCloudSchedulerClient -> %w", err)
	}
	defer c.Close()

	requestHeaders := map[string]string{
		"Content-Type": "application/json",
	}

	payload := &struct {
		ID string `json:"id"`
	}{ID: t.ID}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal -> %w", err)
	}

	oidcToken := &schedulerpb.OidcToken{
		ServiceAccountEmail: fmt.Sprintf("deployer@%s.iam.gserviceaccount.com", projectID),
		Audience:            uri,
	}

	httpOidcToken := &schedulerpb.HttpTarget_OidcToken{
		OidcToken: oidcToken,
	}

	httpTarget := &schedulerpb.HttpTarget{
		Uri:                 fmt.Sprintf("%s/api/v1/measurements", uri),
		HttpMethod:          schedulerpb.HttpMethod_POST,
		Headers:             requestHeaders,
		Body:                requestBody,
		AuthorizationHeader: httpOidcToken,
	}

	jobHttpTarget := &schedulerpb.Job_HttpTarget{
		HttpTarget: httpTarget,
	}

	retryConfig := &schedulerpb.RetryConfig{
		RetryCount: 0,
		MaxRetryDuration: &durationpb.Duration{
			Seconds: 0,
			Nanos:   0,
		},
		MinBackoffDuration: &durationpb.Duration{
			Seconds: 5,
			Nanos:   0,
		},
		MaxBackoffDuration: &durationpb.Duration{
			Seconds: 3600,
			Nanos:   0,
		},
		MaxDoublings: 5,
	}

	job := &schedulerpb.Job{
		Name:        fmt.Sprintf("projects/%s/locations/%s/jobs/%s", projectID, region, t.ID),
		Target:      jobHttpTarget,
		Schedule:    t.Schedule.CronExpression,
		TimeZone:    "UTC",
		RetryConfig: retryConfig,
		AttemptDeadline: &durationpb.Duration{
			Seconds: 180,
			Nanos:   0,
		},
	}

	req := &schedulerpb.CreateJobRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectID, region),
		Job:    job,
	}

	resp, err := c.CreateJob(ctx, req)
	if err != nil {
		return fmt.Errorf("c.CreateJob -> %w", err)
	}

	log.Println(Entry{
		TaskID:    t.ID,
		Severity:  "DEBUG",
		Message:   fmt.Sprintf("%v", resp),
		Component: "scheduler",
		Trace:     trace,
	})

	return nil
}
