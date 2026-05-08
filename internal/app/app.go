package app

import "orquestador-auditor/internal/cli"

func Run() error {
	return cli.RunFromOSArgs()
}
