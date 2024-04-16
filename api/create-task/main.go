package p

import (
	"bytes"
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

	run "cloud.google.com/go/run/apiv2"

	"google.golang.org/api/idtoken"

	runpb "google.golang.org/genproto/googleapis/cloud/run/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	log.SetFlags(0)
	firestoreCollectionName = os.Getenv("FIRESTORE_COLLECTION_NAME")
}

var trace string

func CreateTask(w http.ResponseWriter, r *http.Request) {
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
	case "POST":
		var t *task
		var err error

		counter := 0
		for {
			counter += 1

			t, err = newTask()
			taskID = t.ID
			if err != nil {
				log.Println(Entry{
					TaskID:    taskID,
					Severity:  "ERROR",
					Message:   fmt.Errorf("newTask -> %w", err).Error(),
					Component: "task",
					Trace:     trace,
				})
				continue
			}

			_, err := getTaskFromFirestore(r.Context(), taskID)
			if status.Code(errors.Unwrap(err)) == codes.NotFound {
				break
			}

			if counter == 10 {
				msg := "Unable to find a unique task ID"
				log.Println(Entry{
					TaskID:    taskID,
					Severity:  "ERROR",
					Message:   msg,
					Component: "firestore",
					Trace:     trace,
				})
				sendRespond(w, http.StatusInternalServerError, msg)
				return
			}
		}

		if err := json.NewDecoder(r.Body).Decode(t); err != nil {
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("json.NewDecoder.Decode -> %w", err).Error(),
				Component: "body",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, err.Error())
			return
		}

		receivedPayload, err := json.Marshal(t)
		if err != nil {
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("json.Marshal -> %w", err).Error(),
				Component: "json",
				Trace:     trace,
			})
			sendRespond(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Println(Entry{
			TaskID:    taskID,
			Severity:  "DEBUG",
			Message:   string(receivedPayload),
			Component: "function",
			Trace:     trace,
		})

		if t.Schedule.StartTime.IsZero() && t.Schedule.StopTime.IsZero() {
			t.Type = "one-off_as-soon-as-possible"
		} else if !t.Schedule.StartTime.IsZero() && t.Schedule.StopTime.IsZero() {
			t.Type = "one-off_scheduled"
		} else if t.Schedule.StartTime.IsZero() && !t.Schedule.StopTime.IsZero() {
			t.Type = "recurring_as-soon-as-possible"
		} else {
			t.Type = "recurring_scheduled"
		}
		log.Println(Entry{
			TaskID:    taskID,
			Severity:  "DEBUG",
			Message:   t.Type,
			Component: "function",
			Trace:     trace,
		})

		for _, vantagePoint := range t.VantagePoints {
			t.NumberOfSequence[vantagePoint] = 0
		}

		if err := addTaskToFirestore(r.Context(), t); err != nil {
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("addTaskToFirestore -> %w", err).Error(),
				Component: "firestore",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, err.Error())
			return
		}

		errs := make(chan error, len(t.VantagePoints))
		successes := make(chan string, len(t.VantagePoints))
		var wg sync.WaitGroup
		if t.Type == "one-off_as-soon-as-possible" {
			for _, vantagePoint := range t.VantagePoints {
				wg.Add(1)
				go func(t *task, vantagePoint string) {
					defer wg.Done()
					runClient, err := run.NewServicesClient(r.Context())
					if err != nil {
						msg := fmt.Errorf("%s: %w", vantagePoint, err)
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   msg.Error(),
							Component: "run",
							Trace:     trace,
						})
						errs <- msg
						return
					}
					defer runClient.Close()

					req := &runpb.GetServiceRequest{
						Name: fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, vantagePoint, "measurer"),
					}
					resp, err := runClient.GetService(r.Context(), req)
					if err != nil {
						msg := fmt.Errorf("%s: %w", vantagePoint, err)
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   msg.Error(),
							Component: "run",
							Trace:     trace,
						})
						errs <- msg
						return
					}

					ts, err := idtoken.NewTokenSource(r.Context(), resp.Uri)
					if err != nil {
						msg := fmt.Errorf("%s: %w", vantagePoint, err)
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   msg.Error(),
							Component: "token",
							Trace:     trace,
						})
						errs <- msg
						return
					}

					token, err := ts.Token()
					if err != nil {
						msg := fmt.Errorf("%s: %w", vantagePoint, err)
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   msg.Error(),
							Component: "token",
							Trace:     trace,
						})
						errs <- msg
						return
					}

					httpRequestBody, err := json.Marshal(&struct {
						ID string `json:"id"`
					}{ID: taskID})
					if err != nil {
						msg := fmt.Errorf("%s: %w", vantagePoint, err)
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   msg.Error(),
							Component: "json",
							Trace:     trace,
						})
						errs <- msg
						return
					}

					httpReqPayload := bytes.NewBuffer(httpRequestBody)

					httpReq, err := http.NewRequest(http.MethodPost, resp.Uri+"/api/v1/measurements", httpReqPayload)
					if err != nil {
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   fmt.Errorf("%s: %w", vantagePoint, err).Error(),
							Component: "http",
							Trace:     trace,
						})
						errs <- fmt.Errorf("%s: %w", vantagePoint, err)
						return
					}
					token.SetAuthHeader(httpReq)
					httpReq.Header.Set("X-Cloud-Trace-Context", traceHeader)

					httpClient := &http.Client{}

					go httpClient.Do(httpReq)

					successes <- vantagePoint
				}(t, vantagePoint)
			}

		} else {
			for _, vantagePoint := range t.VantagePoints {
				wg.Add(1)
				go func(t *task, vantagePoint string) {
					defer wg.Done()
					runClient, err := run.NewServicesClient(r.Context())
					if err != nil {
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   fmt.Errorf("%s: %w", vantagePoint, err).Error(),
							Component: "run",
							Trace:     trace,
						})
						errs <- fmt.Errorf("%s: %w", vantagePoint, err)
						return
					}
					defer runClient.Close()

					req := &runpb.GetServiceRequest{
						Name: fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, vantagePoint, "measurer"),
					}
					resp, err := runClient.GetService(r.Context(), req)
					if err != nil {
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   fmt.Errorf("%s: %w", vantagePoint, err).Error(),
							Component: "run",
							Trace:     trace,
						})
						errs <- fmt.Errorf("%s: %w", vantagePoint, err)
						return
					}

					err = createScheduler(r.Context(), projectID, t, vantagePoint, resp.Uri)
					if err != nil {
						log.Println(Entry{
							TaskID:    taskID,
							Severity:  "ERROR",
							Message:   fmt.Errorf("%s: %w", vantagePoint, err).Error(),
							Component: "scheduler",
							Trace:     trace,
						})
						errs <- fmt.Errorf("%s: %w", vantagePoint, err)
						return
					}
					successes <- vantagePoint
				}(t, vantagePoint)
			}
		}

		wg.Wait()
		close(errs)
		close(successes)

		successCount := 0
		for range successes {
			successCount += 1
		}

		if successCount == 0 {
			updateTaskStatus(r.Context(), taskID, "failed")
		}

		errors := make([]error, 0)
		for err := range errs {
			errors = append(errors, err)
		}
		if len(errors) != 0 {
			msg := fmt.Sprintf("Task created, ID: '%s'. However, the following error(s) occurred: %+v", taskID, errors)
			log.Println(Entry{
				TaskID:    taskID,
				Severity:  "ERROR",
				Message:   msg,
				Component: "function",
				Trace:     trace,
			})
			sendRespond(w, http.StatusInternalServerError, msg)
			return
		}

		sendRespond(w, http.StatusCreated, taskID)
		return
	default:
		sendRespond(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
		return
	}
}
