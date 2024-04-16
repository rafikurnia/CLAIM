package api

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/exp/maps"

	"cloud.google.com/go/firestore"

	firebase "firebase.google.com/go"

	"github.com/fatih/structs"

	"github.com/rafikurnia/measurement-measurer/tasks"
)

var firestoreCollectionName string

func getTaskMetadata(ctx context.Context, taskID string) (*tasks.TaskMetadata, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("firebase.NewApp -> %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("app.Firestore -> %w", err)
	}
	defer client.Close()

	dsnap, err := client.Collection(firestoreCollectionName).Doc(taskID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("client.Collection.Get -> %w", err)
	}

	testData, err := tasks.NewTaskMetadata()
	if err != nil {
		return nil, fmt.Errorf("tasks.NewTaskMetadata -> %w", err)
	}

	err = dsnap.DataTo(testData)
	if err != nil {
		return nil, fmt.Errorf("dsnap.DataTo -> %w", err)
	}
	return testData, nil
}

func updateTaskMetadata(ctx context.Context, taskID string, data []firestore.Update) error {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return fmt.Errorf("firebase.NewApp -> %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("app.Firestore -> %w", err)
	}
	defer client.Close()

	_, err = client.Collection(firestoreCollectionName).Doc(taskID).Update(ctx, data)
	if err != nil {
		return fmt.Errorf("client.Collection.Update -> %w", err)
	}

	return nil
}

func updateMeasurementStatus(ctx context.Context, t string, mustDeleteScheduler bool) bool {
	metadata, _ := getTaskMetadata(ctx, t)
	if metadata.NumberOfSequence[os.Getenv("REGION")] > 0 {
		if metadata.Schedule.StopTime.IsZero() || (!metadata.Schedule.StopTime.IsZero() &&
			metadata.Schedule.StopTime.Before(time.Now())) {

			if mustDeleteScheduler {
				deleteScheduler(ctx, t)
			}

			seqStop := 1
			metadata, _ = getTaskMetadata(ctx, t)
			for _, v := range maps.Values(metadata.NumberOfSequence) {
				seqStop = seqStop * v
			}

			if seqStop != 0 && metadata.Status == "running" {
				updateTaskMetadata(ctx, t, []firestore.Update{
					{Path: "Schedule.StopTime", Value: time.Now()},
					{Path: "Status", Value: "finished"},
				})
			}
			return true
		}
	}
	return false
}

func uploadToFirestore(ctx context.Context, taskID string, t *tasks.Task) error {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return fmt.Errorf("firebase.NewApp -> %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("app.Firestore -> %w", err)
	}
	defer client.Close()

	data := structs.Map(t)
	delete(data, "Region")
	delete(data, "Sequence")

	_, err = client.Collection(firestoreCollectionName).Doc(taskID).Collection(t.Region).Doc(strconv.Itoa(t.Sequence)).Set(ctx, data)
	if err != nil {
		return fmt.Errorf("client.Collection.Set -> %w", err)
	}

	return nil
}
