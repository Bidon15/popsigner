// Package main provides the baokey CLI for managing OpenBao-backed keyrings.
package main

import (
	"os"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}

