package deployments

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gh-depdash/internal/githubapi"
)

type Service struct {
	Client githubapi.Client
}

func (s Service) BuildRows(ctx context.Context, owner, repo string, includePlans bool) ([]Row, error) {
	if s.Client == nil {
		return nil, errors.New("deployments service requires client")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	environments, err := s.Client.ListEnvironments(owner, repo)
	if err != nil {
		return nil, err
	}

	filtered := make([]githubapi.Environment, 0, len(environments))
	for _, environment := range environments {
		if !includePlans && isPlanEnvironment(environment.Name) {
			continue
		}
		filtered = append(filtered, environment)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i].Name
		right := filtered[j].Name

		leftPriority, leftBase, leftPlan := environmentSortKey(left)
		rightPriority, rightBase, rightPlan := environmentSortKey(right)

		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if leftBase != rightBase {
			return leftBase < rightBase
		}
		if leftPlan != rightPlan {
			return !leftPlan && rightPlan
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})

	rows := make([]Row, 0, len(filtered))
	for _, environment := range filtered {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		row, err := s.buildRow(owner, repo, environment.Name)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (s Service) buildRow(owner, repo, environment string) (Row, error) {
	deployments, err := s.Client.ListDeployments(owner, repo, environment)
	if err != nil {
		return Row{}, err
	}

	sortDeploymentsNewestFirst(deployments)

	row := Row{Environment: environment}
	var latestAttempt *githubapi.DeploymentStatus

	for _, deployment := range deployments {
		statuses, err := s.Client.ListDeploymentStatuses(owner, repo, deployment.ID)
		if err != nil {
			return Row{}, err
		}
		if len(statuses) == 0 {
			continue
		}

		sortStatusesNewestFirst(statuses)
		current := statuses[0]
		if latestAttempt == nil {
			latestAttempt = &current
		}

		if !isLastKnownGood(current.State) {
			continue
		}

		row.Branch = deployment.Ref
		row.Date = parseTime(deployment.CreatedAt)
		row.Status = current.State
		row.LogURL = current.LogURL
		row.HasSuccess = true
		return row, nil
	}

	if latestAttempt != nil {
		row.Status = latestAttempt.State
		row.LogURL = latestAttempt.LogURL
	}

	return row, nil
}

func sortDeploymentsNewestFirst(deployments []githubapi.Deployment) {
	sort.SliceStable(deployments, func(i, j int) bool {
		left := parseTime(deployments[i].CreatedAt)
		right := parseTime(deployments[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return deployments[i].ID > deployments[j].ID
	})
}

func sortStatusesNewestFirst(statuses []githubapi.DeploymentStatus) {
	sort.SliceStable(statuses, func(i, j int) bool {
		left := parseTime(statuses[i].CreatedAt)
		right := parseTime(statuses[j].CreatedAt)
		return left.After(right)
	})
}

func isPlanEnvironment(name string) bool {
	return strings.HasSuffix(name, "/Plan")
}

func isLastKnownGood(state string) bool {
	switch state {
	case "success", "inactive":
		return true
	default:
		return false
	}
}

func environmentSortKey(name string) (priority int, base string, plan bool) {
	base = name
	plan = isPlanEnvironment(name)
	if plan {
		base = strings.TrimSuffix(name, "/Plan")
	}

	switch strings.ToLower(base) {
	case "development":
		priority = 0
	case "test":
		priority = 1
	case "uat":
		priority = 2
	case "production":
		priority = 3
	default:
		priority = 4
	}

	return priority, strings.ToLower(base), plan
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
