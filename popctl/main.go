// popctl is the command-line interface for POPSigner's control plane.
//
// Use it to manage cryptographic keys, sign data, and organize namespaces
// remotely using your API key.
package main

import "github.com/Bidon15/popsigner/popctl/cmd"

func main() {
	cmd.Execute()
}

