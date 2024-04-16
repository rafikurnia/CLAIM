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
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	log.SetFlags(0)
	firestoreCollectionName = os.Getenv("FIRESTORE_COLLECTION_NAME")
}

var trace string

func CancelTask(w http.ResponseWriter, r *http.Request) {
	var bt, et int64
	bt = time.Now().UnixNano() / int64(time.Millisecond)

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
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
	case "DELETE":
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

		if task.Status == "finished" {
			msg := "The task status is finished"
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "status",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, msg)
			return
		}

		if task.Status == "cancelled" {
			msg := "The task status is cancelled"
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "status",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, msg)
			return
		}

		if task.Status == "failed" {
			msg := "The task status is failed"
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "status",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, msg)
			return
		}

		if task.Type == "one-off_as-soon-as-possible" {
			msg := "The task is a one-off_as-soon-as-possible measurement and thus does not have scheduler to cancel"
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "status",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, msg)
			return
		}

		var wg sync.WaitGroup
		errs := make(chan error, len(task.VantagePoints))
		for _, vantagePoint := range task.VantagePoints {
			wg.Add(1)
			go func(id, vp string) {
				defer wg.Done()

				err := deleteScheduler(r.Context(), id)
				if err != nil {
					errs <- fmt.Errorf("%s: deleteScheduler -> %w", vp, err)
				}
			}(taskID, vantagePoint)
		}
		wg.Wait()

		close(errs)
		errors := make([]error, 0)
		for err = range errs {
			errors = append(errors, err)
		}

		if len(errors) != 0 {
			msg := fmt.Sprintf("the following error(s) occurred: %+v", errors)
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "status",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, msg)
			return
		}

		err = updateTaskStatus(r.Context(), taskID, "cancelled")
		if err != nil {
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("updateTaskStatus -> %w", err).Error(),
				Component: "scheduler",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendRespond(w, http.StatusNoContent, http.StatusText(http.StatusNoContent))
		return
	default:
		sendRespond(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
		return
	}
}
