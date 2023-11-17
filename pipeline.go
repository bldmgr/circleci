package circleci

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	restPipeline          = "api/v2/pipeline?org-slug=gh/%s"
	restPipelineId        = "api/v2/pipeline/%s"
	restPipelineWorkflows = "api/v2/pipeline/%s/workflow"
	restPipelineConfig    = "api/v2/pipeline/%s/config"
)

type listPipelineResponse struct {
	Items             []PipelineItem `json:"items"`
	ContinuationToken string         `json:"next_page_token"`
}

type Trigger struct {
	ReceivedAt time.Time `json:"received_at"`
	Type       string    `json:"type"`
	Actor      Actor     `json:"actor"`
}
type Actor struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type Comiit struct {
	Body    string `json:"body"`
	Subject string `json:"subject"`
}
type Vcs struct {
	OriginRepositoryURL string `json:"origin_repository_url"`
	TargetRepositoryURL string `json:"target_repository_url"`
	Revision            string `json:"revision"`
	ProviderName        string `json:"provider_name"`
	Branch              string `json:"branch"`
	Commit              Comiit `json:"commit"`
}

type Errors struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
type PipelineItem struct {
	ID          string    `json:"id"`
	Errors      []Errors  `json:"errors"`
	ProjectSlug string    `json:"project_slug"`
	UpdatedAt   time.Time `json:"updated_at"`
	Number      int       `json:"number"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	Trigger     Trigger   `json:"trigger"`
	Vcs         Vcs       `json:"vcs"`
}

type Prameters struct {
	PipelineID string `json:"pipeline_id"`
	Parameter  string `json:"parameter"`
	Default    string `json:"default"`
	Type       string `json:"type"`
	Enum       string `json:"enum"`
}

type Job struct {
	Machine       string `json:"machine"`
	Image         string `json:"image"`
	ResourceClass string `json:"resource_class"`
}

func GetPipelineById(ci CI, pipelineId string, output string) (items PipelineItem) {
	var p PipelineItem
	url := fmt.Sprintf(restPipelineId, pipelineId)
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
	} else {
		fmt.Printf("Pipeline Id: %s -> %s \n", p.State, p.CreatedAt)
		fmt.Printf("Pipeline Number: %d -> %s \n", p.Number, p.Vcs.Branch)
	}

	return p
}

type listGetPipelineWorkflowsResponse struct {
	Items             []PipelineWorkflows `json:"items"`
	ContinuationToken string              `json:"next_page_token"`
}

type PipelineWorkflows struct {
	PipelineID     string    `json:"pipeline_id"`
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	ProjectSlug    string    `json:"project_slug"`
	Status         string    `json:"status"`
	StartedBy      string    `json:"started_by"`
	PipelineNumber int       `json:"pipeline_number"`
	CreatedAt      time.Time `json:"created_at"`
	StoppedAt      time.Time `json:"stopped_at"`
	Tag            string    `json:"tag,omitempty"`
}

func GetPipelineWorkflows(ci CI, pipelineId string, output string) (items []PipelineWorkflows) {
	continuation := ""

	get := func() (listResp listGetPipelineWorkflowsResponse, err error) {
		url := fmt.Sprintf(restPipelineWorkflows, pipelineId)

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

	items = make([]PipelineWorkflows, 0)
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
			fmt.Printf("%v: %s -> %s \n", items[i].CreatedAt, items[i].Name, items[i].ID)
		}
	}

	return items
}

type PipelineConfig struct {
	Source              string `json:"source"`
	Compiled            string `json:"compiled"`
	SetupConfig         string `json:"setup-config"`
	CompiledSetupConfig string `json:"compiled-setup-config"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func GetPipelineConfig(ci CI, pipelineId string, output string) (prametersItems []Prameters, jobItems []Job, jsonItems string) {
	var p PipelineConfig
	var w []Prameters
	var j []Job
	url := fmt.Sprintf(restPipelineConfig, pipelineId)
	body, resp, err := ci.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}

	err = json.Unmarshal(body, &p)
	if err != nil {
		fmt.Printf("could not read items from response: %v", err)
	}
	//output = "file"
	if output == "json" {
		fmt.Printf(string(body) + "\n")
	} else if output == "file" {
		title := ""
		jobs := ""
		group := ""

		rf, err := os.ReadFile(".circleci/config.yml")
		if err != nil {
			fmt.Println(err)
			return
		}

		viper.ReadConfig(bytes.NewBuffer(rf))
		version := fmt.Sprint(viper.Get("version"))
		fmt.Println(version)
		text := string(rf)
		fileScanner := bufio.NewScanner(strings.NewReader(text))
		fileScanner.Split(bufio.ScanLines)
		for fileScanner.Scan() {
			s := fileScanner.Text()
			if i := strings.IndexByte(s, ':'); i >= 0 {
				s = s[:i]
			}
			if s == "" {
				jobs = ""
				group = ""
			} else {
				firstCharacter := s[0:1]
				if firstCharacter != "#" {
					if countLeadingSpaces(s) == 0 {
						title = strings.TrimSpace(s)
						if title == "jobs" {
							fmt.Printf("%s: \n", title)
						}
					}
					if countLeadingSpaces(s) == 2 {
						jobs = strings.TrimSpace(s)
						if title == "jobs" {
							fmt.Printf("%s %s:\n", strings.Repeat(" ", 1), jobs)
						}
						jobs_value := fmt.Sprint(viper.Get(title + ".machine"))
						jobs_resource_class := fmt.Sprint(viper.Get(title + ".resource_class"))
						if jobs == "machine" && jobs_value != "<nil>" && "map" != fmt.Sprint(jobs_value[0:3]) {
							fmt.Printf("%s %s: %s\n", strings.Repeat(" ", 3), jobs, jobs_value)
						}
						if jobs == "resource_class" && jobs_resource_class != "<nil>" {
							fmt.Printf("%s %s: %s\n", strings.Repeat(" ", 3), jobs, jobs_resource_class)
						}
					}
					if countLeadingSpaces(s) == 4 {
						group = strings.TrimSpace(s)
						machine_value := fmt.Sprint(viper.Get(title + "." + jobs + ".machine"))
						resource_class := fmt.Sprint(viper.Get(title + "." + jobs + ".resource_class"))
						if group == "machine" && machine_value != "<nil>" && "map" != fmt.Sprint(machine_value[0:3]) {
							fmt.Printf("%s %s: %s\n", strings.Repeat(" ", 3), group, machine_value)
						}
						if group == "resource_class" && resource_class != "<nil>" {
							fmt.Printf("%s %s: %s\n", strings.Repeat(" ", 3), group, resource_class)
						}
					}
					if countLeadingSpaces(s) == 6 {
						value := strings.TrimSpace(s)
						image_value := fmt.Sprint(viper.Get(title + "." + jobs + "." + group + ".image"))
						machine_value := fmt.Sprint(viper.Get(title + "." + jobs + ".machine"))

						if group == "machine" {
							if machine_value != "<nil>" {
								fmt.Printf("%s %s:\n", strings.Repeat(" ", 3), group)
								if image_value != "<nil>" {
									fmt.Printf("%s %s: %s\n", strings.Repeat(" ", 5), value, image_value)
								}

							}
						}
					}
				}
			}
		}
	} else {
		circleci_config := []byte(p.Source)
		viper.SetConfigType("yaml")
		viper.ReadConfig(bytes.NewBuffer(circleci_config))

		group := ""
		title := ""
		value := ""
		j = make([]Job, 0)
		w = make([]Prameters, 0)
		scanner := bufio.NewScanner(strings.NewReader(p.Source))
		for scanner.Scan() {
			s := scanner.Text()
			if i := strings.IndexByte(s, ':'); i >= 0 {
				s = s[:i]
			}
			if s == "" {
				group = ""
				title = ""
			} else {
				firstCharacter := s[0:1]
				if firstCharacter != "#" {
					if countLeadingSpaces(s) == 0 {
						title = s
					} else {
						if countLeadingSpaces(s) == 2 {
							if title == "jobs" {
								job := strings.TrimSpace(s)
								machine := fmt.Sprint(viper.Get(title + "." + job + ".machine"))
								image := fmt.Sprint(viper.Get(title + "." + job + ".image"))
								resource_class := fmt.Sprint(viper.Get(title + "." + job + ".resource_class"))
								j = append(j, Job{
									Machine:       machine,
									Image:         image,
									ResourceClass: resource_class,
								})
							}
						}
						if countLeadingSpaces(s) == 2 {
							if title == "parameters" {
								group = strings.TrimSpace(s)
								value = fmt.Sprint(viper.Get(title + "." + group + ".default"))
								ptype := fmt.Sprint(viper.Get(title + "." + group + ".type"))
								penum := fmt.Sprint(viper.Get(title + "." + group + ".enum"))
								fmt.Println(group, value, ptype, penum)
								w = append(w, Prameters{
									PipelineID: pipelineId,
									Parameter:  group,
									Default:    value,
									Type:       ptype,
									Enum:       penum,
								})
							}
						}
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("error occurred: %v\n", err)
		}
	}

	jsonOut, err := json.Marshal(w)
	if err != nil {
		log.Println(err)
	}
	fullJson := string(jsonOut)

	return w, j, fullJson

}

func countLeadingSpaces(line string) int {
	count := 0
	for _, v := range line {
		if v == ' ' {
			count++
		} else {
			break
		}
	}

	return count
}

type listGetPipeline struct {
	Items             []PipelineItem `json:"items"`
	ContinuationToken string         `json:"next_page_token"`
}

func GetPipeline(ci CI, org string, output string, page int) (items []PipelineItem) {
	continuation := ""
	get := func() (listResp listGetPipeline, err error) {
		url := fmt.Sprintf(restPipeline, org)

		if continuation != "" {
			url += "&page-token=" + continuation
		}

		url += "&mine=false"

		body, resp, err := ci.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Println(url)
			fmt.Println(resp.Status)
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

	items = make([]PipelineItem, 0)
	for i := 0; i < page; i++ {
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

	if output == "xstatus" {
		for i := range items {
			fmt.Printf("%v: %s -> %s --> %s -> %s\n", items[i].Number, items[i].ProjectSlug, items[i].ID, items[i].State, items[i].Trigger.Actor.Login)
		}
	}

	return items
}
