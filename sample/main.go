package main

import (
	"encoding/json"
	"fmt"
	"github.com/bldmgr/circleci"
	setting "github.com/bldmgr/circleci/pkg/config"
	"log"
	"os"
	"strconv"
	"time"
)

func createFile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	f.Close()
}

func appendToFile(line string, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = fmt.Fprintln(f, line)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}

var countryTz = map[string]string{
	"Hungary": "Europe/Budapest",
	"Egypt":   "Africa/Cairo",
	"Canada":  "Canada/Toronto",
}

func timeIn(name string) time.Time {
	loc, err := time.LoadLocation(countryTz[name])
	if err != nil {
		panic(err)
	}
	return time.Now().In(loc)
}

func main() {

	loadedConfig := setting.SetConfigYaml()

	ci, err := circleci.New(loadedConfig.Host, loadedConfig.Token, loadedConfig.Project)
	if err != nil {
		panic(err)
	}

	status := circleci.Me(ci)
	fmt.Printf("Connection to %s was successful -> %t \n", loadedConfig.Host, status)
	createFile("test.json")
	w := circleci.GetPipeline(ci, "bldmgr", "json", 2)
	testdate := "2024-06-20T17:42:50.528Z"
	formattedDate, err := time.Parse(time.RFC3339, testdate)
	fmt.Println(formattedDate)
	utc := time.Now().UTC().Format("15:04")
	hun := timeIn("Hungary").Format("15:04")
	eg := timeIn("Ottawa").Format("15:04")
	fmt.Println(utc, hun, eg)
	getWorkflow(ci, w)
}

func getWorkflow(ci circleci.CI, pipeline []circleci.PipelineItem) {
	alldata := make([]circleci.AllData, 0)
	workflowpipeline := make([]circleci.WorkflowPipeline, 0)
	var returnDataSet []circleci.JobDataSteps
	var returnEnvConfig []circleci.JobDataEnvironment
	var orbs []circleci.ViperSub
	var parameters []circleci.ViperSub
	//for p := range pipeline {
	//	pipelineId := pipeline[p].ID
	//pipelineId := "13aa1fc1-adde-469c-be61-619047b2782e" // tangerine
	pipelineId := "aae2b1ef-e7d7-46e7-9a8e-71effdb17af7"
	fmt.Printf("Pipeline Id: %s \n", pipelineId)

	workflows := circleci.GetPipelineWorkflows(ci, pipelineId, "none")
	for w := range workflows {
		//truncated := truncate.Truncator(payload[i].Vcs.Revision, 9, truncate.CutStrategy{})
		testWorkflowId := "ff0f7a34-b837-4b21-b3a1-a564bb37b1f8"
		fmt.Printf("--> Workflow Id: %s \n", workflows[w].ID)
		var jobs []circleci.WorkflowItem = circleci.GetWorkflowJob(ci, testWorkflowId, "json", "i.data", "i.token")

		for j := range jobs {
			jd := circleci.GetJobDetails(ci, strconv.Itoa(jobs[j].JobNumber), "gh", "Cloud", "janus-rails", "")
			fmt.Println(jd.Parallelism)
			fmt.Printf("-->> Checking %v %s status: %s \n", jobs[j].JobNumber, jobs[j].Name, jobs[j].Status)
			// job loop
			returnDataSet, returnEnvConfig, orbs, parameters = circleci.GetConfigWithWorkflow(ci, jobs, workflows, j, w, "data")
			log.Printf("Config %v", returnEnvConfig[0].Sha)
			for o := range orbs {
				fmt.Println(orbs[o].Name)
			}
			for q := range parameters {
				fmt.Println(parameters[q].Name)
			}

			workflowpipeline = append(workflowpipeline, circleci.WorkflowPipeline{
				JobNumber:    jobs[j].JobNumber,
				Id:           jobs[j].Id,
				StartedAt:    jobs[j].StartedAt,
				Name:         jobs[j].Name,
				ProjectSlug:  jobs[j].ProjectSlug,
				Status:       jobs[j].Status,
				Type:         jobs[j].Type,
				StoppedAt:    jobs[j].StoppedAt,
				JobDataSteps: returnDataSet,
			})
		}
		alldata = append(alldata, circleci.AllData{
			PipelineID:       workflows[w].PipelineID,
			ID:               workflows[w].ID,
			Name:             workflows[w].Name,
			ProjectSlug:      workflows[w].ProjectSlug,
			Status:           workflows[w].Status,
			StartedBy:        workflows[w].StartedBy,
			PipelineNumber:   workflows[w].PipelineNumber,
			CreatedAt:        workflows[w].CreatedAt,
			StoppedAt:        workflows[w].StoppedAt,
			Tag:              workflows[w].Tag,
			WorkflowPipeline: workflowpipeline,
		})

		// workflow loop
		workflowpipeline = workflowpipeline[:0]
		fmt.Println("output json")
		out, err := json.Marshal(alldata)
		if err != nil {
			log.Println(err)
		}
		// remove first and last character from string.
		m := string(out[1:])

		appendToFile(m[:len(m)-1], "test.json")
	}
	// pipeline loop

	fmt.Println("new pipeline scanned")
	alldata = alldata[:0]
	//}

	//	}
}
