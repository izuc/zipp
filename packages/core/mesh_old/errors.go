package mesh_old

import "github.com/cockroachdb/errors"

var (
	// ErrNotBootstrapped is triggered when somebody tries to issue a Payload before the Mesh is fully bootstrapped.
	ErrNotBootstrapped = errors.New("mesh not bootstrapped")
	// ErrParentsInvalid is returned when one or more parents of a block is invalid.
	ErrParentsInvalid = errors.New("one or more parents is invalid")
)
