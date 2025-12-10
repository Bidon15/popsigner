package banhbaoring

import "errors"

// TODO(01B): Implement all errors below

// Sentinel errors
var (
	ErrMissingBaoAddr   = errors.New("banhbaoring: BaoAddr is required")
	ErrMissingBaoToken  = errors.New("banhbaoring: BaoToken is required")
	ErrMissingStorePath = errors.New("banhbaoring: StorePath is required")

	ErrKeyNotFound      = errors.New("banhbaoring: key not found")
	ErrKeyExists        = errors.New("banhbaoring: key already exists")
	ErrKeyNotExportable = errors.New("banhbaoring: key is not exportable")

	ErrBaoConnection  = errors.New("banhbaoring: connection failed")
	ErrBaoAuth        = errors.New("banhbaoring: authentication failed")
	ErrBaoSealed      = errors.New("banhbaoring: OpenBao is sealed")
	ErrBaoUnavailable = errors.New("banhbaoring: OpenBao unavailable")

	ErrSigningFailed    = errors.New("banhbaoring: signing failed")
	ErrInvalidSignature = errors.New("banhbaoring: invalid signature")
	ErrUnsupportedAlgo  = errors.New("banhbaoring: unsupported algorithm")
	ErrStorePersist     = errors.New("banhbaoring: persist failed")
	ErrStoreCorrupted   = errors.New("banhbaoring: store corrupted")
)

