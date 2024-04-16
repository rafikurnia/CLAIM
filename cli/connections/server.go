package connections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/rafikurnia/measurement-cli/tasks"
	"github.com/rafikurnia/measurement-cli/utils/log"
)

const (
	// server       = "http://localhost:50000"
	// server       = "http://35.219.115.68:50000"
	server       = "https://measurement-platform-2o0gyw1z.ew.gateway.dev"
	basePath     = "api/v1"
	resourceName = "measurements"
)

// Data structure for response on HTTP calls
type httpResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var logger = log.GetLogger("server")

func SendTask(task *tasks.Task) {
	endpoint := fmt.Sprintf("%s/%s/%s", server, basePath, resourceName)

	dataForLog, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("Sent data: %s", string(dataForLog))

	taskJSON, err := json.Marshal(task)
	if err != nil {
		logger.Error(err)
		return
	}

	reqBody := bytes.NewBuffer(taskJSON)

	resp, err := http.Post(endpoint, "application/json", reqBody)

	if err != nil {
		logger.Errorf("An Error Occured %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("An Error Occured %v", err)
		return
	}

	response := &httpResponse{}
	err = json.Unmarshal(body, response)
	if err != nil {
		logger.Error(err)
		return
	}

	if resp.StatusCode == 400 {
		if strings.Contains(string(response.Message), "Schedule or time zone is invalid") {
			logger.Errorf("Error 400: bad cron expression: '%s'", task.Schedule.CronExpression)
			return
		}

		logger.Errorf("Error 400: bad request: %s", string(response.Message))
		return
	} else if resp.StatusCode != 201 {
		logger.Errorf("Error %d: %s", resp.StatusCode, string(response.Message))
		return
	}

	logger.Infof("Task ID: %s", string(response.Message))
}

func GetStatus(taskID string) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", server, basePath, resourceName, taskID)

	resp, err := http.Get(endpoint)
	if err != nil {
		logger.Fatalf("Cannot connect to server: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err)
	}

	response := &httpResponse{}
	err = json.Unmarshal(body, response)
	if err != nil {
		logger.Error(err)
		return
	}

	if resp.StatusCode == 404 {
		logger.Errorf("Error 404: cannot find a task with ID: %s", taskID)
		return
	} else if resp.StatusCode == 400 {
		logger.Errorf("Error 400: invalid task ID: %s", taskID)
		return
	} else if resp.StatusCode != 200 {
		logger.Errorf("Error %d: %s", resp.StatusCode, string(response.Message))
		return
	}

	logger.Infof("Status: %s", string(body))
}

func CancelTask(taskID string) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", server, basePath, resourceName, taskID)

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		logger.Fatalf("Cannot connect to server: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		logger.Errorf("Error 404: cannot find a task with ID: %s", taskID)
		return
	} else if resp.StatusCode == 400 {
		logger.Errorf("Error 400: a task with ID: %s is neither running nor scheduled", taskID)
		return
	} else if resp.StatusCode != 204 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error(err)
			return
		}

		response := &httpResponse{}
		err = json.Unmarshal(body, response)
		if err != nil {
			logger.Error(err)
			return
		}

		logger.Errorf("Error %d: %s", resp.StatusCode, string(response.Message))
		return
	}

	logger.Infof("The task with id: %s is canceled", taskID)
}

func GetResults(taskID string) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s/results", server, basePath, resourceName, taskID)

	resp, err := http.Get(endpoint)
	if err != nil {
		logger.Fatalf("Cannot connect to server: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err)
		return
	}

	response := &httpResponse{}
	err = json.Unmarshal(body, response)
	if err != nil {
		logger.Error(err)
		return
	}

	if resp.StatusCode == 404 {
		logger.Errorf("Error 404: cannot find a task with ID: %s", taskID)
		return
	} else if resp.StatusCode == 400 {
		logger.Errorf("Error 400: a task with ID %s is not in finished state", taskID)
		return
	} else if resp.StatusCode == 202 {
		logger.Info("The results are not ready yet. Please try again later.")
		return
	} else if resp.StatusCode != 200 {
		logger.Errorf("Error %d: %s", resp.StatusCode, string(response.Message))
		return
	}

	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, []byte(response.Message), "", "  ")
	if error != nil {
		logger.Error(err)
		return
	}

	err = os.WriteFile(fmt.Sprintf("./%s.json", taskID), prettyJSON.Bytes(), 0644)
	if err != nil {
		logger.Error(err)
		return
	}

	logger.Infof("The results are saved to ./%s.json", taskID)
}
