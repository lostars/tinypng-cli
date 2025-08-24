package main

import "tinypng-cli/cmd"

var (
	Version = "none"
)

func main() {
	cmd.Execute(Version)
}
