package config

import (
	b64 "encoding/base64"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const (
	homeEnvVar       = "CIRCLE_HOME"
	hostEnvVar       = "CIRCLE_HOSTNAME"
	tokenEnvVar      = "CIRCLE_TOKEN"
	projectEnvVar    = "CIRCLE_PROJECT"
	pipelineIdEnvVar = "CIRCLE_PIPELINEID"
)

type ConfigYaml struct {
	Host       string
	Token      string
	Project    string
	PipelineID string
	Type       string
}

func SetConfigYaml() *ConfigYaml {

	ciHome := defaultCiHome()
	if os.Getenv(hostEnvVar) == "" {
		viper.SetConfigType("yaml")
		viper.AddConfigPath(ciHome)
		viper.SetConfigName("/ci")
		err := viper.ReadInConfig()

		if err != nil {
			fmt.Printf("%v", err)
		}

		conf := &ConfigYaml{}
		err = viper.Unmarshal(conf)
		if err != nil {
			fmt.Printf("unable to decode into config struct, %v", err)
		}

		return &ConfigYaml{
			Host:       fmt.Sprintf("%v", viper.Get("circle_hostname")),
			Project:    fmt.Sprintf("%v", viper.Get("circle_project")),
			Token:      strings.TrimSpace(LetsDecrypt(fmt.Sprintf("%v", viper.Get("circle_token")))),
			PipelineID: fmt.Sprintf("%v", viper.Get("circle_pipelineId")),
			Type:       fmt.Sprintf("%v", "yamlVar"),
		}
	} else {
		return &ConfigYaml{
			Host:       os.Getenv(hostEnvVar),
			Token:      strings.TrimSpace(LetsDecrypt(fmt.Sprintf("%v", os.Getenv(tokenEnvVar)))),
			Project:    os.Getenv(projectEnvVar),
			PipelineID: os.Getenv(pipelineIdEnvVar),
			Type:       fmt.Sprintf("%v", "osEnvVar"),
		}
	}
}

func defaultCiHome() string {
	if home := os.Getenv(homeEnvVar); home != "" {
		return home
	}
	homeEnvPath := os.Getenv("HOME")

	return filepath.Join(homeEnvPath, ".")
}

func LetsDecrypt(p string) string {
	sDec, _ := b64.StdEncoding.DecodeString(p)
	return string(sDec)
}
