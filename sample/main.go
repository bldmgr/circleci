package main

import (
	"encoding/json"
	"fmt"
	"github.com/bldmgr/circleci"
	setting "github.com/bldmgr/circleci/pkg/config"
	"log"
	"os"
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

func main() {

	loadedConfig := setting.SetConfigYaml()

	ci, err := circleci.New(loadedConfig.Host, loadedConfig.Token, loadedConfig.Project)
	if err != nil {
		panic(err)
	}

	status := circleci.Me(ci)
	fmt.Printf("Connection to %s was successful -> %t \n", loadedConfig.Host, status)
	createFile("test.json")
	w := circleci.GetPipeline(ci, "Cloud", "web", 1)
	getWorkflow(ci, w)
}

func getWorkflow(ci circleci.CI, pipeline []circleci.PipelineItem) {
	alldata := make([]circleci.AllData, 0)
	workflowpipeline := make([]circleci.WorkflowPipeline, 0)
	var returnDataSet []circleci.JobDataSteps
	//for p := range pipeline {
	//pipelineId := pipeline[p].ID
	pipelineId := "0ae71535-f159-4486-ba08-f478b6394d77"
	fmt.Printf("Pipeline Id: %s \n", pipelineId)

	workflows := circleci.GetPipelineWorkflows(ci, pipelineId, "none")
	for w := range workflows {
		//truncated := truncate.Truncator(payload[i].Vcs.Revision, 9, truncate.CutStrategy{})

		fmt.Printf("--> Workflow Id: %s \n", workflows[w].ID)
		var jobs []circleci.WorkflowItem = circleci.GetWorkflowJob(ci, workflows[w].ID, "none", "i.data", "i.token")
		for j := range jobs {
			fmt.Printf("-->> Checking %v %s status: %s \n", jobs[j].JobNumber, jobs[j].Name, jobs[j].Status)
			// job loop
			returnDataSet = circleci.GetConfigWithWorkflow(ci, jobs, workflows, j, w, "")
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

}
