package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/rafikurnia/measurement-measurer/tasks"
	"github.com/rafikurnia/measurement-measurer/utils"
	"github.com/rafikurnia/measurement-measurer/utils/logger"

	hstat "github.com/tcnksm/go-httpstat"

	"golang.org/x/exp/maps"
)

var trace string

func runMeasurement(ctx *gin.Context) {
	var bt, mbt, met, et int64
	bt = time.Now().UnixNano() / int64(time.Millisecond)

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID != "" {
		traceHeader := ctx.Request.Header.Get("X-Cloud-Trace-Context")
		traceParts := strings.Split(traceHeader, "/")
		if len(traceParts) > 0 && len(traceParts[0]) > 0 {
			trace = fmt.Sprintf("projects/%s/traces/%s", projectID, traceParts[0])
		}
	}

	benchmarkID := ctx.Request.URL.Query().Get("benchmark_id")
	benchmarkSeq := ctx.Request.URL.Query().Get("benchmark_sequence")

	bid, _ := strconv.ParseInt(benchmarkID, 10, 64)
	bseq, _ := strconv.ParseInt(benchmarkSeq, 10, 64)

	task := &struct {
		ID string `json:"id"`
	}{}

	defer func() {
		et = time.Now().UnixNano() / int64(time.Millisecond)

		bench := &benchmark{
			BeginningTime:            bt,
			MeasurementBeginningTime: mbt,
			MeasurementEndingTime:    met,
			EndingTime:               et,
		}

		inJson, _ := json.Marshal(bench)

		log.Println(logger.Entry{
			// TaskID:            task.ID,
			Severity:          "INFO",
			Message:           string(inJson),
			Component:         "benchmark",
			BenchmarkID:       bid,
			BenchmarkSequence: bseq,
			Trace:             trace,
		})
	}()

	defer func() {
		if r := recover(); r != nil {
			var err error
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}

			log.Println(logger.Entry{
				// TaskID:            task.ID,
				Severity:          "CRITICAL",
				Message:           err.Error(),
				Component:         "panic",
				BenchmarkID:       bid,
				BenchmarkSequence: bseq,
				Trace:             trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}()

	if err := ctx.ShouldBindBodyWith(task, binding.JSON); err != nil {
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "ERROR",
			Message:   fmt.Errorf("ctx.ShouldBindBodyWith -> %w", err).Error(),
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if task.ID == "" {
		err := errors.New("Missing task ID")
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "ERROR",
			Message:   err.Error(),
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusBadRequest, err.Error())
		return
	}

	metadata, _ := getTaskMetadata(ctx, task.ID)

	mustDeleteScheduler := true
	if metadata.Type == "one-off_as-soon-as-possible" {
		mustDeleteScheduler = false
	}

	if metadata.Status == "finished" {
		if mustDeleteScheduler {
			deleteScheduler(ctx, task.ID)
		}
		msg := "The measurement is finished"
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "INFO",
			Message:   msg,
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusOK, msg)
		return
	}

	if !metadata.Schedule.StartTime.IsZero() &&
		metadata.Schedule.StartTime.After(time.Now()) {
		msg := fmt.Sprintf("The StartTime is in the future: %v", metadata.Schedule.StartTime)
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "INFO",
			Message:   msg,
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusOK, msg)
		return
	}

	if updateMeasurementStatus(ctx, task.ID, mustDeleteScheduler) {
		msg := "The job is done"
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "INFO",
			Message:   msg,
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusOK, msg)
		return
	}

	seqStart := 0
	metadata, _ = getTaskMetadata(ctx, task.ID)
	for _, v := range maps.Values(metadata.NumberOfSequence) {
		seqStart = seqStart * v
	}

	taskResult, err := tasks.NewTask()
	if err != nil {
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "WARN",
			Message:   err.Error(),
			Component: "task",
			Trace:     trace,
		})
	}

	taskResult.MeasurementStartTime = time.Now()
	mbt = taskResult.MeasurementStartTime.UnixNano() / int64(time.Millisecond)

	if seqStart == 0 && metadata.Status == "scheduled" {
		updateTaskMetadata(ctx, task.ID, []firestore.Update{
			{Path: "Schedule.StartTime", Value: taskResult.MeasurementStartTime},
			{Path: "Status", Value: "running"},
		})
	}

	if metadata.Probe == "ping" || metadata.Probe == "traceroute" || metadata.Probe == "curl" {
		var command string
		if metadata.Probe == "ping" {
			command = fmt.Sprintf("ping -c 1 %s", metadata.Arguments)
		} else if metadata.Probe == "traceroute" {
			command = fmt.Sprintf("traceroute %s", metadata.Arguments)
		} else {
			command = fmt.Sprintf("curlt %s", metadata.Arguments)
		}

		cmd := exec.Command("sh", "-c", command)
		defer func() {
			switch err := cmd.Process.Kill(); err {
			case nil:
				log.Println(logger.Entry{
					// TaskID:    task.ID,
					Severity:  "INFO",
					Message:   fmt.Sprintf("The %s process has been terminated.", metadata.Probe),
					Component: "api",
					Trace:     trace,
				})

			case os.ErrProcessDone:
				err = nil

			default:
				log.Println(logger.Entry{
					// TaskID:    task.ID,
					Severity:  "ERROR",
					Message:   fmt.Errorf("cmd.Process.Kill -> %w", err).Error(),
					Component: "api",
					Trace:     trace,
				})
				return
			}
		}()

		// Get the pipe for stdout
		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("cmd.StdoutPipe -> %w", err).Error(),
				Component: "api",
				Trace:     trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, err.Error())
			return
		}

		// Set stderr to also being sent to stdout
		cmd.Stderr = cmd.Stdout

		// Start executing the command
		err = cmd.Start()
		if err != nil {
			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("cmd.Start -> %w", err).Error(),
				Component: "api",
				Trace:     trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, err.Error())
			return
		}

		scanner := bufio.NewScanner(cmdReader)

		var storage string
		for scanner.Scan() {
			t := scanner.Text()
			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "INFO",
				Message:   t,
				Component: "api",
				Trace:     trace,
			})
			storage += fmt.Sprintf("%s\n", t)
		}
		cmd.Wait()

		taskResult.Result = storage

	} else if metadata.Probe == "httpstat" {
		if !strings.HasPrefix(metadata.Arguments, "http://") &&
			!strings.HasPrefix(metadata.Arguments, "https://") {

			err := errors.New("The arguments must contain URL starts with either 'http://' or 'https://'.")

			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "ERROR",
				Message:   err.Error(),
				Component: "api",
				Trace:     trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, err.Error())
			return
		}

		req, err := http.NewRequest("GET", strings.ReplaceAll(metadata.Arguments, "\n", ""), nil)
		if err != nil {
			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("http.NewRequest -> %w", err).Error(),
				Component: "api",
				Trace:     trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, err.Error())
			return
		}

		// The code below is mostly obtained from:
		// https://medium.com/@deeeet/trancing-http-request-latency-in-golang-65b2463f548c

		// Create a httpstat powered context
		var result hstat.Result
		c := hstat.WithHTTPStat(req.Context(), &result)
		req = req.WithContext(c)

		// Send request by default HTTP client
		client := http.DefaultClient
		res, err := client.Do(req)
		if err != nil {
			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("client.Do -> %w", err).Error(),
				Component: "api",
				Trace:     trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
			log.Println(logger.Entry{
				// TaskID:    task.ID,
				Severity:  "ERROR",
				Message:   fmt.Errorf("io.Copy -> %w", err).Error(),
				Component: "api",
				Trace:     trace,
			})
			utils.Throws(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		res.Body.Close()

		outputs := fmt.Sprintf(
			"DNS lookup: %d ms\n"+
				"TCP connection: %d ms\n"+
				"TLS handshake: %d ms\n"+
				"Server processing: %d ms\n"+
				"Content transfer: %d ms\n",
			int(result.DNSLookup/time.Millisecond),
			int(result.TCPConnection/time.Millisecond),
			int(result.TLSHandshake/time.Millisecond),
			int(result.ServerProcessing/time.Millisecond),
			int(result.StartTransfer/time.Millisecond),
		)

		taskResult.Result = outputs
	}

	taskResult.Sequence = metadata.NumberOfSequence[os.Getenv("REGION")] + 1
	taskResult.MeasurementStopTime = time.Now()
	met = taskResult.MeasurementStopTime.UnixNano() / int64(time.Millisecond)

	err = uploadToFirestore(ctx, task.ID, taskResult)
	if err != nil {
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "ERROR",
			Message:   fmt.Errorf("uploadToFirestore -> %w", err).Error(),
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	updateTaskMetadata(ctx, task.ID, []firestore.Update{
		{Path: fmt.Sprintf("NumberOfSequence.%s", os.Getenv("REGION")), Value: taskResult.Sequence},
	})

	data, err := json.Marshal(taskResult)
	if err != nil {
		log.Println(logger.Entry{
			// TaskID:    task.ID,
			Severity:  "ERROR",
			Message:   fmt.Errorf("json.Marshal -> %w", err).Error(),
			Component: "api",
			Trace:     trace,
		})
		utils.Throws(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	updateMeasurementStatus(ctx, task.ID, mustDeleteScheduler)
	utils.Throws(ctx, http.StatusOK, string(data))
}
