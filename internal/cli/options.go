package cli

import (
	"errors"
	"flag"
)

type Options struct {
	Repo         string
	IncludePlans bool
	Verbose      bool
	JSON         bool
}

func Parse(args []string) (Options, error) {
	var opts Options

	fs := flag.NewFlagSet("gh-depdash", flag.ContinueOnError)
	fs.SetOutput(flag.CommandLine.Output())
	fs.BoolVar(&opts.Verbose, "verbose", false, "")
	fs.BoolVar(&opts.IncludePlans, "plans", false, "")
	fs.BoolVar(&opts.JSON, "json", false, "")
	fs.StringVar(&opts.Repo, "repo", "", "")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	switch fs.NArg() {
	case 0:
	case 1:
		if opts.Repo == "" {
			opts.Repo = fs.Arg(0)
		} else if opts.Repo != fs.Arg(0) {
			return Options{}, errors.New("repo provided twice with different values")
		}
	default:
		return Options{}, errors.New("too many arguments")
	}

	return opts, nil
}
