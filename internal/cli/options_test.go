package cli

import "testing"

func TestParseDefaults(t *testing.T) {
	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if opts != (Options{}) {
		t.Fatalf("Parse() = %#v, want zero Options", opts)
	}
}

func TestParseVerboseAndPlans(t *testing.T) {
	opts, err := Parse([]string{"--verbose", "--plans"})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if !opts.Verbose {
		t.Fatalf("Parse() Verbose = false, want true")
	}
	if !opts.IncludePlans {
		t.Fatalf("Parse() IncludePlans = false, want true")
	}
}

func TestParseJSONAndRepo(t *testing.T) {
	opts, err := Parse([]string{"--json", "--repo", "octo/example"})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if !opts.JSON {
		t.Fatalf("Parse() JSON = false, want true")
	}
	if opts.Repo != "octo/example" {
		t.Fatalf("Parse() Repo = %q, want %q", opts.Repo, "octo/example")
	}
}

func TestParsePositionalRepo(t *testing.T) {
	opts, err := Parse([]string{"owner/repo"})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if opts.Repo != "owner/repo" {
		t.Fatalf("Parse() Repo = %q, want %q", opts.Repo, "owner/repo")
	}
}

func TestParseTooManyArguments(t *testing.T) {
	if _, err := Parse([]string{"owner/repo", "extra"}); err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}
}

func TestParseSameRepoViaFlagAndPositional(t *testing.T) {
	opts, err := Parse([]string{"--repo", "owner/repo", "owner/repo"})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if opts.Repo != "owner/repo" {
		t.Fatalf("Parse() Repo = %q, want %q", opts.Repo, "owner/repo")
	}
}

func TestParseRejectsUnknownFlag(t *testing.T) {
	if _, err := Parse([]string{"--nope"}); err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}
}
