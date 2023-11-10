package circleci

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// ServerInfo contains the information needed to connect to a Circle server
type ServerInfo struct {
	Host, Token, Project string
}

// Client is the interface which allows interacting with an IQ server
type Client interface {
	CurlRequest(method, endpoint string) (*http.Request, error)
	NewRequest(method, endpoint string, payload io.Reader) (*http.Request, error)
	Get(endpoint string) ([]byte, *http.Response, error)
	Post(endpoint string, payload io.Reader) ([]byte, *http.Response, error)
	Info() ServerInfo
}

// DefaultClient provides an HTTP wrapper with optimized for communicating with a Circle server
type DefaultClient struct {
	ServerInfo
	Debug bool
}

// NewRequest created an http.Request object based on an endpoint and fills in basic auth
func (s *DefaultClient) NewRequest(method, endpoint string, payload io.Reader) (request *http.Request, err error) {
	url := fmt.Sprintf("%s/%s", s.Host, endpoint)
	request, err = http.NewRequest(method, url, payload)
	if err != nil {
		return
	}

	request.SetBasicAuth(s.Token, "")
	//if payload != nil {
	request.Header.Set("Content-Type", "application/json")
	//}

	return
}

func (s *DefaultClient) CurlRequest(method, endpoint string) (request *http.Request, err error) {
	url := fmt.Sprintf("%s/%s", s.Host, endpoint)
	request, err = http.NewRequest(method, url, nil)
	if err != nil {
		return
	}
	request.Header.Set("Circle-Token", s.Token)
	request.Header.Set("Content-Type", "application/json")

	return
}

// Do performs an http.Request and reads the body if StatusOK
func (s *DefaultClient) Do(request *http.Request) (body []byte, resp *http.Response, err error) {
	if s.Debug {
		dump, _ := httputil.DumpRequest(request, true)
		log.Println("debug: http request:")
		log.Printf("%q\n", dump)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err = client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// TODO: Trying to decide if this is a horrible idea or just kinda bad
	//if resp.StatusCode == http.StatusOK {
	//	body, err = ioutil.ReadAll(resp.Body)
	//	fmt.Println(string(body))
	//	return
	//}

	err = errors.New(resp.Status)
	body, err = ioutil.ReadAll(resp.Body)
	// debug fmt.Println(string(body))
	return
}

func (s *DefaultClient) http(method, endpoint string, payload io.Reader) ([]byte, *http.Response, error) {
	request, err := s.NewRequest(method, endpoint, payload)
	if err != nil {
		return nil, nil, err
	}

	return s.Do(request)
}

// Info return information about the Nexus server
func (s *DefaultClient) Info() ServerInfo {
	return ServerInfo{s.Host, s.Token, s.Project}
}

// Get performs an HTTP GET against the indicated endpoint
func (s *DefaultClient) Get(endpoint string) ([]byte, *http.Response, error) {
	return s.http(http.MethodGet, endpoint, nil)
}

// Post performs an HTTP POST against the indicated endpoint
func (s *DefaultClient) Post(endpoint string, payload io.Reader) ([]byte, *http.Response, error) {
	return s.http(http.MethodPost, endpoint, payload)
}
