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

type RepositoryPermissions struct {
	Push bool `json:"push"`
}

type Repository struct {
	Name        string                `json:"name"`
	FullName    string                `json:"full_name"`
	Description *string               `json:"description"`
	Private     bool                  `json:"private"`
	Visibility  string                `json:"visibility"`
	Permissions RepositoryPermissions `json:"permissions"`
	UpdatedAt   string                `json:"updated_at"`
}

type RepositoryPage struct {
	Repositories []Repository
	HasMore      bool
}

type Client interface {
	ListEnvironments(owner, repo string) ([]Environment, error)
	ListDeployments(owner, repo, environment string) ([]Deployment, error)
	ListDeploymentStatuses(owner, repo string, deploymentID int64) ([]DeploymentStatus, error)
	ListRepositories(page, perPage int) (RepositoryPage, error)
}
