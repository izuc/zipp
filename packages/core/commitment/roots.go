package commitment

import (
	"golang.org/x/crypto/blake2b"

	"github.com/izuc/zipp.foundation/core/model"
	"github.com/izuc/zipp.foundation/ds/types"
	"github.com/izuc/zipp.foundation/serializer/byteutils"
)

type Roots struct {
	model.Immutable[Roots, *Roots, roots] `serix:"0"`
}

type roots struct {
	MeshRoot          types.Identifier `serix:"0"`
	StateMutationRoot types.Identifier `serix:"1"`
	ActivityRoot      types.Identifier `serix:"4"`
	StateRoot         types.Identifier `serix:"2"`
	ManaRoot          types.Identifier `serix:"3"`
}

func NewRoots(meshRoot, stateMutationRoot, activityRoot, stateRoot, manaRoot types.Identifier) (newRoots *Roots) {
	return model.NewImmutable[Roots](&roots{
		MeshRoot:          meshRoot,
		StateMutationRoot: stateMutationRoot,
		ActivityRoot:      activityRoot,
		StateRoot:         stateRoot,
		ManaRoot:          manaRoot,
	})
}

func (r *Roots) ID() (id types.Identifier) {
	branch1Hashed := blake2b.Sum256(byteutils.ConcatBytes(r.M.MeshRoot.Bytes(), r.M.StateMutationRoot.Bytes()))
	branch2Hashed := blake2b.Sum256(byteutils.ConcatBytes(r.M.StateRoot.Bytes(), r.M.ManaRoot.Bytes()))
	rootHashed := blake2b.Sum256(byteutils.ConcatBytes(branch1Hashed[:], branch2Hashed[:]))

	return rootHashed
}

func (r *Roots) MeshRoot() (meshRoot types.Identifier) {
	return r.M.MeshRoot
}

func (r *Roots) StateMutationRoot() (stateMutationRoot types.Identifier) {
	return r.M.StateMutationRoot
}

func (r *Roots) StateRoot() (stateRoot types.Identifier) {
	return r.M.StateRoot
}

func (r *Roots) ManaRoot() (manaRoot types.Identifier) {
	return r.M.ManaRoot
}

func (r *Roots) ActivityRoot() (activityRoot types.Identifier) {
	return r.M.ActivityRoot
}
