package p

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"

	"google.golang.org/api/iterator"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	log.SetFlags(0)
}

func GetResults(w http.ResponseWriter, r *http.Request) {
	var bt, et int64
	bt = time.Now().UnixNano() / int64(time.Millisecond)

	var trace string
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	firestoreCollectionName := os.Getenv("FIRESTORE_COLLECTION_NAME")
	if projectID != "" {
		traceHeader := r.Header.Get("X-Cloud-Trace-Context")
		traceParts := strings.Split(traceHeader, "/")
		if len(traceParts) > 0 && len(traceParts[0]) > 0 {
			trace = fmt.Sprintf("projects/%s/traces/%s", projectID, traceParts[0])
		}
	}

	benchmarkID := r.URL.Query().Get("benchmark_id")
	benchmarkSeq := r.URL.Query().Get("benchmark_sequence")

	bid, _ := strconv.ParseInt(benchmarkID, 10, 64)
	bseq, _ := strconv.ParseInt(benchmarkSeq, 10, 64)

	var taskID string

	defer func() {
		et = time.Now().UnixNano() / int64(time.Millisecond)

		bench := &benchmark{
			BeginningTime: bt,
			EndingTime:    et,
			DeltaETandBT:  et - bt,
		}

		inJson, _ := json.Marshal(bench)

		log.Println(Entry{
			TaskID:            taskID,
			Severity:          "INFO",
			Message:           string(inJson),
			Component:         "benchmark",
			BenchmarkID:       bid,
			BenchmarkSequence: bseq,
			Trace:             trace,
		})
	}()

	switch r.Method {
	case "GET":
		urls := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(urls) < 4 {
			msg := "empty parameter"
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "parameter",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, msg)
			return
		}

		taskID = strings.TrimSpace(urls[3])
		if taskID == "" {
			msg := "empty parameter"
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "parameter",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, msg)
			return
		}

		app, err := firebase.NewApp(r.Context(), nil)
		if err != nil {
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("firebase.NewApp -> %w", err).Error(),
				Component: "firebase",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, err.Error())
			return
		}

		client, err := app.Firestore(r.Context())
		if err != nil {
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("app.Firestore -> %w", err).Error(),
				Component: "firestore",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer client.Close()

		dsnap, err := client.Collection(firestoreCollectionName).Doc(taskID).Get(r.Context())
		if err != nil {
			if status.Code(err) == codes.NotFound {
				log.Println(Entry{
					TaskID:    taskID,
					Severity:  "ERROR",
					Message:   fmt.Errorf("client.Get -> %w", err).Error(),
					Component: "firestore",
					Trace:     trace,
				})
				sendRespond(w, http.StatusNotFound, err.Error())
				return
			}

			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("client.Get -> %w", err).Error(),
				Component: "firestore",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, err.Error())
			return
		}

		metadata := dsnap.Data()

		jobStatus := metadata["Status"]
		if jobStatus == "scheduled" {
			sendRespond(w, http.StatusAccepted, http.StatusText(http.StatusAccepted))
			return
		}

		count := 0
		iter := client.Collection(firestoreCollectionName).Doc(taskID).Collections(r.Context())

		reg := make(map[string]map[int]map[string]interface{})
		for {
			collRef, err := iter.Next()
			if err == iterator.Done {
				break
			}
			count += 1
			if err != nil {
				log.Println(Entry{
					TaskID:    taskID,
					Severity:  "ERROR",
					Message:   fmt.Errorf("iter.Next -> %w", err).Error(),
					Component: "firestore",
					Trace:     trace,
				})
				sendRespond(w, http.StatusInternalServerError, err.Error())
				return
			}

			nextIter := client.Collection(firestoreCollectionName).Doc(taskID).Collection(collRef.ID).Documents(r.Context())

			seq := make(map[int]map[string]interface{})
			for {
				doc, err := nextIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Println(Entry{
						TaskID:    taskID,
						Severity:  "ERROR",
						Message:   fmt.Errorf("nextIter.Next -> %w", err).Error(),
						Component: "firestore",
						Trace:     trace,
					})
					sendRespond(w, http.StatusInternalServerError, err.Error())
					return
				}

				docName := doc.Ref.ID

				valInt, err := strconv.ParseInt(docName, 10, 32)
				if err != nil {
					log.Println(Entry{
						TaskID:    taskID,
						Severity:  "ERROR",
						Message:   fmt.Errorf("strconv.ParseInt -> %w", err).Error(),
						Component: "firestore",
						Trace:     trace,
					})
					sendRespond(w, http.StatusInternalServerError, err.Error())
					return
				}

				seq[int(valInt)] = doc.Data()
			}

			reg[collRef.ID] = seq
		}

		metadata["ID"] = taskID
		metadata["Results"] = reg

		b, _ := json.Marshal(metadata)

		sendRespond(w, http.StatusOK, string(b))
		return
	default:
		sendRespond(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
		return
	}
}
