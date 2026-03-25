package main

import (
	"os"

	"github.com/gstephens/poeusage/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
