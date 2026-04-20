package deployments

import "time"

type Row struct {
	Environment string
	Branch      string
	SHA         string
	Date        time.Time
	Status      string
	LogURL      string
	HasSuccess  bool
}
