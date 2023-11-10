package circleci

type CI interface {
	Client
}

type ciClient struct {
	DefaultClient
}

func New(host, token, project string) (CI, error) {
	ci := new(ciClient)
	ci.Host = host
	ci.Token = token
	ci.Project = project

	return ci, nil
}
