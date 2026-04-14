package app

import "orquestador-auditor/internal/cli"

func Run(version string) error {
	_ = version
	return cli.RunFromOSArgs()
}
