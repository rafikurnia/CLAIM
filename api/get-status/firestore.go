package p

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go"
)

var firestoreCollectionName string

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
