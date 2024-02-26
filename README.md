# circleci

The CircleCI API may be used to make API calls to retrieve detailed information about users, jobs, workflows and pipelines. There are currently two supported API versions:

`CircleCI API v1.1 and v2 are supported and generally available.`


`go get -u github.com/bldmgr/circleci`

```golang
package main

import (
	"fmt"
	"github.com/bldmgr/circleci"
	setting "github.com/bldmgr/circleci/pkg/config"
)

func main() {

	loadedConfig := setting.SetConfigYaml()

	ci, err := circleci.New(loadedConfig.Host, loadedConfig.Token, loadedConfig.Project)
	if err != nil {
		panic(err)
	}

	status := circleci.Me(ci)
	fmt.Printf("Connection to %s was successful -> %t \n", "host", status)
}

```