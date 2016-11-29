package core

import "fmt"

type version struct {
	Major, Minor, Patch int
	Label               string
	Name                string
}

var Version = version{1, 2, 3, "dev", "huilixiang"}

var Build string

func (v version) String() string {
	if v.Label != "" {
		return fmt.Sprintf("ccevent version %d.%d.%d-%s \"%s\"\nGit commit hash: %s", v.Major, v.Minor, v.Patch, v.Label, v.Name, Build)
	} else {
		return fmt.Sprintf("ccevent version %d.%d.%d \"%s\"\nGit commit hash: %s", v.Major, v.Minor, v.Patch, v.Name, Build)
	}
}
