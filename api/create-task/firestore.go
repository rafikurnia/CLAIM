package p

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/fatih/structs"
)

var firestoreCollectionName string

func addTaskToFirestore(ctx context.Context, t *task) error {
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
	delete(data, "ID")

	log.Println(Entry{
		TaskID:    t.ID,
		Severity:  "DEBUG",
		Message:   fmt.Sprintf("%v", data),
		Component: "firestore",
		Trace:     trace,
	})

	_, err = client.Collection(firestoreCollectionName).Doc(t.ID).Set(ctx, data)
	if err != nil {
		return fmt.Errorf("client.Set -> %w", err)
	}

	return nil
}

func getTaskFromFirestore(ctx context.Context, taskID string) (*task, error) {
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
		return nil, fmt.Errorf("client.Get -> %w", err)
	}

	testData, err := newTask()
	if err != nil {
		return nil, fmt.Errorf("newTask -> %w", err)
	}

	err = dsnap.DataTo(testData)
	if err != nil {
		return nil, fmt.Errorf("dsnap.DataTo -> %w", err)
	}
	return testData, nil
}

func updateTaskStatus(ctx context.Context, taskID, status string) error {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return fmt.Errorf("firebase.NewApp -> %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("app.Firestore -> %w", err)
	}
	defer client.Close()

	_, err = client.Collection(firestoreCollectionName).Doc(taskID).Update(ctx,
		[]firestore.Update{
			{
				Path:  "Status",
				Value: status,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("client.Update -> %w", err)
	}

	return nil
}
