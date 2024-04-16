package api

import (
	"context"
	"fmt"
	"os"

	scheduler "cloud.google.com/go/scheduler/apiv1"
	schedulerpb "google.golang.org/genproto/googleapis/cloud/scheduler/v1"
)

func deleteScheduler(ctx context.Context, taskID string) error {
	c, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		return fmt.Errorf("scheduler.NewCloudSchedulerClient -> %w", err)
	}
	defer c.Close()

	req := &schedulerpb.DeleteJobRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/jobs/%s", os.Getenv("GOOGLE_CLOUD_PROJECT"), os.Getenv("REGION"), taskID),
	}
	err = c.DeleteJob(ctx, req)
	if err != nil {
		return fmt.Errorf("c.DeleteJob -> %w", err)
	}

	return nil
}
