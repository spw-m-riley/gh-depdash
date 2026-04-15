package githubapi

type Environment struct {
	Name string `json:"name"`
}

type Deployment struct {
	ID        int64  `json:"id"`
	Ref       string `json:"ref"`
	SHA       string `json:"sha"`
	CreatedAt string `json:"created_at"`
}

type DeploymentStatus struct {
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	LogURL    string `json:"log_url"`
}

type Client interface {
	ListEnvironments(owner, repo string) ([]Environment, error)
	ListDeployments(owner, repo, environment string) ([]Deployment, error)
	ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]DeploymentStatus, error)
}
