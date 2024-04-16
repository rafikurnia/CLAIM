package p

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	log.SetFlags(0)
	firestoreCollectionName = os.Getenv("FIRESTORE_COLLECTION_NAME")
}

var trace string

func GetStatus(w http.ResponseWriter, r *http.Request) {
	var bt, et int64
	bt = time.Now().UnixNano() / int64(time.Millisecond)

	var traceHeader string
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID != "" {
		traceHeader = r.Header.Get("X-Cloud-Trace-Context")
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

		task, err := getTaskFromFirestore(r.Context(), taskID)
		if err != nil {
			if status.Code(errors.Unwrap(err)) == codes.NotFound {
				log.Println(Entry{
					TaskID:    taskID,
					Severity:  "ERROR",
					Message:   fmt.Errorf("getTaskFromFirestore -> %w", err).Error(),
					Component: "firestore",
					Trace:     trace,
				})
				sendRespond(w, http.StatusNotFound, err.Error())
				return
			}

			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("getTaskFromFirestore -> %w", err).Error(),
				Component: "firestore",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendRespond(w, http.StatusOK, task.Status)
		return
	default:
		sendRespond(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
		return
	}
}
