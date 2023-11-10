package circleci

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	restWorkflowJob = "api/v2/workflow/%s/job"
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
