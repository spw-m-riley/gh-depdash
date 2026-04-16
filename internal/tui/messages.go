package tui

import (
	"gh-depdash/internal/githubapi"
	"gh-depdash/internal/output"
)

type repoPageLoadedMsg struct {
	repos   []githubapi.Repository
	hasMore bool
}

type repoPageFailedMsg struct {
	err string
}

type moreReposLoadedMsg struct {
	repos   []githubapi.Repository
	hasMore bool
}

type moreReposFailedMsg struct {
	err string
}

type deploymentsLoadedMsg struct {
	rows            []output.ViewRow
	partialFailures []string
}

type deploymentsFatalErrorMsg struct {
	err string
}
