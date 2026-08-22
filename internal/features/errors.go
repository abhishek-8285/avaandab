package features

import "errors"

// ErrUnknownFeature is returned by Set for keys absent from the Catalog.
var ErrUnknownFeature = errors.New("features: unknown feature key")
