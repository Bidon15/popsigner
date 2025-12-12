// POPSigner secp256k1 Plugin
//
// This OpenBao plugin provides native secp256k1 signing operations.
// Keys never leave the vault boundary.
package main

import (
	"log"
	"os"

	"github.com/openbao/openbao/sdk/v2/plugin"

	"github.com/Bidon15/popsigner/plugin/secp256k1"
)

func main() {
	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: secp256k1.Factory,
	}); err != nil {
		log.Printf("plugin shutting down: %v", err)
		os.Exit(1)
	}
}
