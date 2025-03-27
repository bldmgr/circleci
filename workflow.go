package circleci

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	restWorkflowJob = "api/v2/workflow/%s/job"
	restGetParallel = "api/v2/workflow/%s/job/%s/parallel-runs/1"
)

type listAssetsResponse struct {
	Items             []WorkflowItem `json:"items"`
	ContinuationToken string         `json:"next_page_token"`
}

type WorkflowItem struct {
	JobNumber   int    `json:"job_number"`
	Id          string `json:"id"`
	StartedAt   string `json:"started_at"`
	Name        string `json:"name"`
	ProjectSlug string `json:"project_slug"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	StoppedAt   string `json:"stopped_at"`
}

func GetJobParallel(ci CI, jobId string, vsc string, namespace string, project string, output string) (items JobDetails) {
	var p JobDetails
	url := fmt.Sprintf(restGetParallel, "d0c7ccea-144a-411e-b505-359ebfb296ef", "21692")
	body, resp, err := ci.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}

	err = json.Unmarshal(body, &p)
	if err != nil {
		fmt.Printf("could not read items from response: %v", err)
	}

	if output == "json" {
		fmt.Println("GetJobParallel")
		fmt.Printf(string(body) + "\n")
		fmt.Println("GetJobParallel done")
	}

	return p
}

func GetWorkflowJob(ci CI, workflowId string, output string, data string, token string) (items []WorkflowItem) {
	continuation := ""

	get := func() (listResp listAssetsResponse, err error) {
		url := fmt.Sprintf(restWorkflowJob, workflowId)

		if continuation != "" {
			url += "&continuationToken=" + continuation
		}

		body, resp, err := ci.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			return
		}

		err = json.Unmarshal(body, &listResp)
		if err != nil {
			fmt.Printf("could not read items from response: %v", err)
		}

		if output == "json" {
			fmt.Printf(string(body) + "\n")
		}

		return
	}

	items = make([]WorkflowItem, 0)
	for {
		resp, err := get()
		if err != nil {
			return items
		}

		items = append(items, resp.Items...)
		if resp.ContinuationToken == "" {
			break
		}

		continuation = resp.ContinuationToken
	}

	if output == "status" {
		for i := range items {
			fmt.Printf("Workflow Job: %s Status -> %s \n", items[i].Name, items[i].Status)
		}
	}

	return items
}
