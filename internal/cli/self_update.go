package cli

import (
	"flag"
	"fmt"
	"runtime"

	"orquestador-auditor/internal/update"
)

func runSelfUpdate(args []string) error {
	fs := flag.NewFlagSet("self-update", flag.ContinueOnError)
	var repo string
	var version string
	var check bool
	fs.StringVar(&repo, "repo", "victo/orquestador_auditor", "GitHub repo slug")
	fs.StringVar(&version, "version", "latest", "Release tag to install, or latest")
	fs.BoolVar(&check, "check", false, "Only print the latest available version")
	if err := fs.Parse(args); err != nil {
		return err
	}

	updater := update.New(repo)
	if version == "latest" {
		latest, err := updater.LatestVersion()
		if err != nil {
			return err
		}
		if check {
			fmt.Println(latest)
			return nil
		}
		version = latest
	}

	target, err := updater.Apply(version)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		fmt.Printf("Downloaded updated binary to %s. Replace the current executable after this process exits.\n", target)
		return nil
	}
	fmt.Printf("Updated binary in place: %s\n", target)
	return nil
}
