package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Println("mdit", version)
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, "usage: mdit <file.md|folder>")
	os.Exit(1)
}
