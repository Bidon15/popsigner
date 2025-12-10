package secp256k1

import "time"

// TODO(03B): Implement types

type keyEntry struct {
	PrivateKey []byte    `json:"private_key"`
	PublicKey  []byte    `json:"public_key"`
	Exportable bool      `json:"exportable"`
	CreatedAt  time.Time `json:"created_at"`
	Imported   bool      `json:"imported"`
}

