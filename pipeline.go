package circleci

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
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

type CommitInfo struct {
	Body    string `json:"body"`
	Subject string `json:"subject"`
}
type Vcs struct {
	OriginRepositoryURL string     `json:"origin_repository_url"`
	TargetRepositoryURL string     `json:"target_repository_url"`
	Revision            string     `json:"revision"`
	ProviderName        string     `json:"provider_name"`
	Branch              string     `json:"branch"`
	Commit              CommitInfo `json:"commit"`
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
	}

	if output == "status" {
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

func GetConfigWithWorkflow(ci CI, jobs []WorkflowItem, workflows []PipelineWorkflows, j int, w int, output string) (returnData []JobDataSteps, returnEnvConfig []JobDataEnvironment, orbs []ViperSub, parameters []ViperSub) {
	var p PipelineConfig

	url := fmt.Sprintf(restPipelineConfig, workflows[w].PipelineID)
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

	circleciSource := []byte(p.Source)
	configCompiled := []byte(p.Compiled)
	orbs = processParms(circleciSource, "orbs")
	parameters = processParms(circleciSource, "parameters")

	project, vcs, namespace := formatProjectSlug(workflows[w].ProjectSlug)
	returnDataSet, returnEnvConfig := processJobs(ci, jobs[j].Name, jobs[j].JobNumber, project, namespace, vcs, output, configCompiled)

	return returnDataSet, returnEnvConfig, orbs, parameters
}

func formatProjectSlug(projectSlug string) (project string, vcs string, namespace string) {
	// Split vcs/namespace/project
	out, project := filepath.Split(projectSlug)
	s := strings.TrimRight(out, "/")
	x, namespace := filepath.Split(s)
	vcs = strings.TrimRight(x, "/")

	return project, vcs, namespace
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
	} else if output == "source" {
		circleci_source := p.Source
		fmt.Printf(circleci_source)
	} else if output == "compiled" {
		circleci_compiled := p.Compiled
		fmt.Printf(circleci_compiled)
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

type watcherCmd struct {
	host           string
	token          string
	namespace      string
	pipelineId     string
	action         string
	pipelineNumber string
	createDate     string
}

func (cmd *watcherCmd) run(ci CI, org string, output string, maxPage int) (items []PipelineItem) {
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
	for i := 0; i < maxPage; i++ {
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

type Cache struct {
	Key   string
	Value map[string]any
}

func NewCache(v *viper.Viper, key string) *Cache {

	return &Cache{
		Key:   key,
		Value: v.GetStringMap(key),
	}
}

func loopJobs(jobs []interface{}) {
	for _, job := range jobs {

		switch v := job.(type) {
		case string:
			fmt.Printf("---------- %v ----------\n", v)
			//processJobs(fmt.Sprintf("%v.steps", v))
		default:
			jobMap := job.(map[string]interface{})
			for jobName, jobValue := range jobMap {

				switch v := jobValue.(type) {
				case string:
					fmt.Printf("---------- %v ----------\n", v)
					//processJobs(fmt.Sprintf("%v.steps", v))
				default:
					fmt.Println("----------", jobName, "----------")
					//processJobs(fmt.Sprintf("%v.steps", jobName))
				}
			}
		}
	}
}

func processWorkflows(jobName string) {
	v := viper.Sub("workflows")
	if v == nil { // Sub returns nil if the key cannot be found
		panic("workflows cache configuration not found")
	}
	fmt.Println("*****************************")
	fmt.Println("processing the workflows ... ...")
	fmt.Println("*****************************")
	keys := v.AllKeys()
	workflowName := fmt.Sprintf("%v.jobs", jobName)
	for i := 0; i < len(keys); i++ {
		if strings.Contains(keys[i], ".jobs") {
			if keys[i] == workflowName {
				fmt.Println("================ ", keys[i], " ================")
				var jobs = v.Get(keys[i]).([]interface{})
				loopJobs(jobs)
				fmt.Println("================================")
			}
		}
	}
	fmt.Println("*****************************")
	fmt.Println("finish workflows ... ...")
	fmt.Println("*****************************")
}

func processParms(circleciConfig []byte, viperSub string) []ViperSub {

	viperItems := make([]ViperSub, 0)
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer(circleciConfig))
	v := viper.Sub(viperSub)
	if v == nil { // Sub returns nil if the key cannot be found
		return nil
	}
	keys := v.AllKeys()
	for i := 0; i < len(keys); i++ {

		if viperSub == "parameters" {

			if strings.Contains(keys[i], ".type") {
				var paramType = v.Get(keys[i]).(string)
				nameKey := strings.FieldsFunc(keys[i], func(r rune) bool {
					return r == '.'
				})
				viperItems = append(viperItems, ViperSub{
					Name: nameKey[0],
					Type: paramType,
				})
			}
		} else if viperSub == "orbs" {
			//var data interface{}
			f := viper.Sub("orbs").AllSettings()
			for key, value := range f {

				switch value.(type) {
				case string:
					viperItems = append(viperItems, ViperSub{
						Name: key,
						Type: value.(string),
					})

				case map[string]interface{}:
					newVersion := fmt.Sprintf("%s@embedded", key)
					viperItems = append(viperItems, ViperSub{
						Name: newVersion,
						Type: viperSub,
					})
					for key2, rvalue := range value.(map[string]interface{}) {
						if key2 == "orbs" {
							for key3, rvalue2 := range rvalue.(map[string]interface{}) {
								fmt.Println("Orb:", key3)
								fmt.Println("Version: ", rvalue2)
								orbVersion := v.Get(key3).(string)
								viperItems = append(viperItems, ViperSub{
									Name: key3,
									Type: orbVersion,
								})
							}
						}
					}
				}
			}
			fmt.Printf("done")
		}

		//	ak := f.AllKeys()
		//	fmt.Println(ak)
		//	for _, ak := range ak {
		//		fmt.Println("Key: ", ak)
		//		fmt.Println("Value: ", f.Get(ak))
		//		fmt.Println("Type: ", f.Get(ak).(map[string]interface{})["version"])
		//}
		//data = viper.GetStringMap()
		//switch v := data.(type) {
		//case map[string]int:
		//	fmt.Println("Type: map[string]int", v)
		//	fmt.Println("Total Keys:", f.Get(keys[i]).(string))

		//case map[string]string:
		//	fmt.Println("Type: map[string]string", v)
		//	fmt.Println("Total Keys:", f.Get(keys[i]).(string))

		//case map[string]interface{}:
		//	fmt.Println("Type: map[string]interface{}", v)
		//	fmt.Println("Total Keys:", f.Get(keys[i]))
		//	var orbVersion = ""
		//	viperItems = append(viperItems, ViperSub{
		//		Name: keys[i],
		//		Type: orbVersion,
		//	})
		//}
		//	}
		//} else {
		//viperItems = append(viperItems, ViperSub{
		//Name: keys[i],
		//Type: viperSub,
		//})
		//}
	}

	return viperItems
}

func processJobs(ci CI, workflowName string, jobNumber int, projectName string, namespace string, vsc string, output string, configCompiled []byte) (Steps []JobDataSteps, Env []JobDataEnvironment) {
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer(configCompiled))

	ghSha := ""
	getSteps := func(jobsSteps []interface{}, sum int) (Steps []JobDataSteps) {
		dataSteps := make([]JobDataSteps, 0)

		data := ""
		data_name := ""
		data_command := ""
		data_path := ""
		data_key := ""
		data_when := ""

		for _, steps := range jobsSteps {
			sum++
			switch v := steps.(type) {
			case string:
				data_name = v
				if v == "checkout" {
					if output == "data" {
						data = string(GetJobData(ci, strconv.Itoa(jobNumber), vsc, namespace, projectName, strconv.Itoa(sum), ""))
					}
				}
			default:
				stepsMap := steps.(map[string]interface{})
				for stepsName, stepsValue := range stepsMap {
					data_command = ""
					data_path = ""
					data_key = ""
					data_when = ""
					data_name = stepsName
					switch v := stepsValue.(type) {
					case string:
						data_name = v
						if output == "data" {
							data = string(GetJobData(ci, strconv.Itoa(jobNumber), vsc, namespace, projectName, strconv.Itoa(sum), ""))
						}
					default:
						jobDetails := stepsValue.(map[string]interface{})
						for key, value := range jobDetails {
							if key == "command" {
								data_command = fmt.Sprintf("%v", value)
							}
							if key == "path" {
								data_path = fmt.Sprintf("%v", value)
							}
							if key == "when" {
								data_when = fmt.Sprintf("%v", value)
							}
							if key == "key" {
								data_key = fmt.Sprintf("%v", value)
							}
						}
					}
					if output == "data" {
						data = string(GetJobData(ci, strconv.Itoa(jobNumber), vsc, namespace, projectName, strconv.Itoa(sum), ""))
					}
				}
			}
			dataSteps = append(dataSteps, JobDataSteps{
				ID:      strconv.Itoa(sum),
				Name:    data_name,
				Command: data_command,
				Key:     data_key,
				Path:    data_path,
				When:    data_when,
				Output:  data,
			})
		}

		return dataSteps
	}

	dataSteps := make([]JobDataSteps, 0)
	dataEnvironment := make([]JobDataEnvironment, 0)
	data := string(GetJobData(ci, strconv.Itoa(jobNumber), vsc, namespace, projectName, "0", ""))
	jobHost := string(data) + "\n"
	outAgent, outRunner, outVm, outImage, outVolume := parseVariables(jobHost, "Build-agent version ", "Launch-agent version ", "Using volume:", "default", "  using image ")
	dataSteps = append(dataSteps, JobDataSteps{
		ID:      "0",
		Name:    "Spin up environment",
		Command: "",
		Key:     "",
		Path:    "",
		When:    "",
		Output:  data,
	})

	data = string(GetJobData(ci, strconv.Itoa(jobNumber), vsc, namespace, projectName, "99", ""))
	jobEnvironment := string(data) + "\n"
	dataSteps = append(dataSteps, JobDataSteps{
		ID:      "99",
		Name:    "Preparing environment variables",
		Command: "",
		Key:     "",
		Path:    "",
		When:    "",
		Output:  data,
	})
	dataReader := strings.NewReader(jobEnvironment)
	scanner := bufio.NewScanner(dataReader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "CIRCLE_SHA1") == true {
			ghSha = strings.Replace(line, "  CIRCLE_SHA1=", "", -1)
		}
	}

	dataEnvironment = append(dataEnvironment, JobDataEnvironment{
		Sha:        ghSha,
		HostAgent:  outAgent,
		HostImage:  outImage,
		HostVM:     outVm,
		HostVolume: outVolume,
		HostRunner: outRunner,
	})

	v := viper.Sub("jobs")
	if v != nil { // Sub returns nil if the key cannot be found

		keys := v.AllKeys()
		for i := 0; i < len(keys); i++ {
			sum := 100
			if strings.Contains(keys[i], fmt.Sprintf("%v.steps", workflowName)) {
				var jobsSteps = v.Get(keys[i]).([]interface{})
				items := getSteps(jobsSteps, sum)
				for i := range items {
					dataSteps = append(dataSteps, JobDataSteps{
						ID:      items[i].ID,
						Name:    items[i].Name,
						Command: items[i].Command,
						Key:     items[i].Key,
						Path:    items[i].Path,
						When:    items[i].When,
						Output:  items[i].Output,
					})
				}
			}
		}
	}

	return dataSteps, dataEnvironment
}

func removeText(data string, start string, end string, number int) (out string) {
	format := strings.Replace(data, start, "", number)
	out = strings.Replace(format, end, "", number)
	return out
}

func parseVariables(data string, agent string, runner string, volume string, vm string, image string) (outAgent string, outRunner string, outVm string, outImage string, outVolume string) {
	theReader := strings.NewReader(data)
	scanner := bufio.NewScanner(theReader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, agent) == true {
			outAgent = strings.Replace(line, agent, "", -1)
		}
		if strings.Contains(line, runner) == true {
			outRunner = strings.Replace(line, runner, "", -1)
		}
		if strings.Contains(line, vm) == true {
			outVm = removeText(line, "VM '", "' has been created", -1)
		}
		if strings.Contains(line, volume) == true {
			outVolume = strings.Replace(line, volume, "", -1)
		}
		if strings.Contains(line, image) == true {
			outImage = strings.Replace(line, image, "", -1)
		}
	}

	return outAgent, outRunner, outVm, outImage, outVolume
}

type JobDataEnvironment struct {
	Sha            string   `json:"sha"`
	HostType       string   `json:"host_type"`
	HostClass      string   `json:"host_class"`
	HostImage      string   `json:"host_image"`
	HostVM         string   `json:"host_vm"`
	HostVolume     string   `json:"host_volume"`
	HostAgent      string   `json:"host_agent"`
	HostRunner     string   `json:"host_runner"`
	ExternalInputs []string `json:"external_inputs"`
	Orbs           []string `json:"orbs"`
	Parameters     []string `json:"parameters"`
}

type JobDataSteps struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Key     string `json:"key"`
	Path    string `json:"path"`
	When    string `json:"when"`
	Output  string `json:"output"`
}

type WorkflowPipeline struct {
	JobNumber    int            `json:"job_number"`
	Id           string         `json:"id"`
	StartedAt    string         `json:"started_at"`
	Name         string         `json:"name"`
	ProjectSlug  string         `json:"project_slug"`
	Status       string         `json:"status"`
	Type         string         `json:"type"`
	StoppedAt    string         `json:"stopped_at"`
	JobDataSteps []JobDataSteps `json:"job_data_steps"`
}

type AllData struct {
	PipelineID       string             `json:"pipeline_id"`
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	ProjectSlug      string             `json:"project_slug"`
	Status           string             `json:"status"`
	StartedBy        string             `json:"started_by"`
	PipelineNumber   int                `json:"pipeline_number"`
	CreatedAt        time.Time          `json:"created_at"`
	StoppedAt        time.Time          `json:"stopped_at"`
	Tag              string             `json:"tag,omitempty"`
	WorkflowPipeline []WorkflowPipeline `json:"workflow_pipeline"`
}

type ViperSub struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
