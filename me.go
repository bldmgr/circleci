package circleci

import (
	"net/http"
)

const (
	restMe = "api/v2/me"
)

func Me(ci CI) (_ bool) {
	_, resp, err := ci.Get(restMe)
	return err == nil && resp.StatusCode == http.StatusOK
}
