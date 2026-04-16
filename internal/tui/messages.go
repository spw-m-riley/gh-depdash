package tui

import "gh-depdash/internal/deployments"

type repoPageLoadedMsg struct {
	repos []string
}

type repoPageFailedMsg struct {
	err string
}

type deploymentsLoadedMsg struct {
	rows            []deployments.Row
	partialFailures []string
}

type deploymentsPartialFailureMsg struct {
	err string
}

type deploymentsFatalErrorMsg struct {
	err string
}
