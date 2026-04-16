package tui

import "gh-depdash/internal/output"

type repoPageLoadedMsg struct {
	repos []string
}

type repoPageFailedMsg struct {
	err string
}

type deploymentsLoadedMsg struct {
	rows            []output.ViewRow
	partialFailures []string
}

type deploymentsFatalErrorMsg struct {
	err string
}
