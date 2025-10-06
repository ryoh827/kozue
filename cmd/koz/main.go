package main

import (
	"os"

	"github.com/ryoh827/kozue/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
