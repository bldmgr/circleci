package circleci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	restGetJobDetails   = "api/v2/project/%s/%s/%s/job/%s"
	restGetJobArtifacts = "api/v2/project/%s/%s/artifacts"
	restGetTestMetadata = "api/v2/project/%s/%s/%s/%s/tests"
	restGetJobData      = "api/v1.1/project/%s/%s/%s/%s/output/%s/0?file=true"
)

type JobDetails struct {
	WebURL  string `json:"web_url"`
	Project struct {
		ExternalURL string `json:"external_url"`
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		ID          string `json:"id"`
	} `json:"project"`
	ParallelRuns []struct {
		Index  int    `json:"index"`
		Status string `json:"status"`
	} `json:"parallel_runs"`
	StartedAt      time.Time `json:"started_at"`
	LatestWorkflow struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"latest_workflow"`
	Name     string `json:"name"`
	Executor struct {
		ResourceClass string `json:"resource_class"`
		Type          string `json:"type"`
	} `json:"executor"`
	Parallelism int    `json:"parallelism"`
	Status      string `json:"status"`
	Number      int    `json:"number"`
	Pipeline    struct {
		ID string `json:"id"`
	} `json:"pipeline"`
	Duration  int           `json:"duration"`
	CreatedAt time.Time     `json:"created_at"`
	Messages  []interface{} `json:"messages"`
	Contexts  []struct {
		Name string `json:"name"`
	} `json:"contexts"`
	Organization struct {
		Name string `json:"name"`
	} `json:"organization"`
	QueuedAt  time.Time `json:"queued_at"`
	StoppedAt time.Time `json:"stopped_at"`
}

type listTestMetadata struct {
	Items             []TestMetadata `json:"items"`
	ContinuationToken string         `json:"next_page_token"`
}

type TestMetadata struct {
	Classname string  `json:"classname"`
	File      string  `json:"file"`
	Name      string  `json:"name"`
	Result    string  `json:"result"`
	Message   string  `json:"message"`
	RunTime   float64 `json:"run_time"`
	Source    string  `json:"source"`
}

type Artifacts struct {
	Items             []ArtifactsItem `json:"items"`
	ContinuationToken string          `json:"next_page_token"`
}
type ArtifactsItem struct {
	NodeIndex int    `json:"node_index"`
	Path      string `json:"path"`
	URL       string `json:"url"`
}

func GetJobsArtifacts(ci CI, jobId string, project string, output string) (items []ArtifactsItem) {
	continuation := ""

	get := func() (listResp Artifacts, err error) {
		url := fmt.Sprintf(restGetJobArtifacts, project, jobId)

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
			fmt.Printf(string(body))
		}

		return
	}

	items = make([]ArtifactsItem, 0)
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

	return items
}

func GetJobDetails(ci CI, jobId string, vsc string, namespace string, project string, output string) (items JobDetails) {
	var p JobDetails
	url := fmt.Sprintf(restGetJobDetails, vsc, namespace, project, jobId)
	body, resp, err := ci.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}

	err = json.Unmarshal(body, &p)
	if err != nil {
		fmt.Printf("could not read items from response: %v", err)
	}

	if output == "json" {
		fmt.Printf(string(body) + "\n")

	}

	return p
}

func GetTestMetadata(ci CI, jobId string, vsc string, namespace string, project string, output string, page int) (items []TestMetadata) {
	continuation := ""

	get := func() (listResp listTestMetadata, err error) {
		url := fmt.Sprintf(restGetTestMetadata, vsc, namespace, project, jobId)

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
			fmt.Printf(string(body))
		}

		return
	}

	items = make([]TestMetadata, 0)
	for i := 0; i < page; i++ {
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
	}

	return items
}

func GetJobData(ci CI, jobId string, vsc string, namespace string, project string, step string, output string) (t string) {

	url := fmt.Sprintf(restGetJobData, vsc, namespace, project, jobId, step)

	body, resp, err := ci.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}

	if output == "data" {
		fmt.Printf(string(body) + "\n")
	}
	output = string(body) + "\n"

	return output
}
