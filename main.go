package main

import (
	"os"

	"github.com/xZhad/pomo/internal/cli"
)

func main() { os.Exit(cli.Run(os.Args[1:], os.Stdout)) }
