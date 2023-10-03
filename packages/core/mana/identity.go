package mana

import (
	"fmt"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/izuc/zipp.foundation/core/marshalutil"
	"github.com/mr-tron/base58"
)

// IDFromStr decodes and returns an ID from base58.
func IDFromStr(idStr string) (iID identity.ID, err error) {
	iID = identity.ID{}
	if idStr == "" {
		return
	}
	bytes, err := base58.Decode(idStr)
	if err != nil {
		err = fmt.Errorf("could not decode ID: %s, from base58: %w", idStr, err)
		return
	}
	copy(iID[:], bytes)
	return
}

// IDFromPubKey returns the ID from the given public key.
func IDFromPubKey(pubKey string) (iID identity.ID, err error) {
	iID = identity.ID{}
	if pubKey == "" {
		return
	}
	bytes, err := base58.Decode(pubKey)
	if err != nil {
		err = fmt.Errorf("could not decode public key: %s, from base58: %w", pubKey, err)
		return
	}
	_identity, err := identity.Parse(marshalutil.New(bytes))
	if err != nil {
		err = fmt.Errorf("could not parse public key: %s, %w", pubKey, err)
		return
	}

	idBytes, err := _identity.ID().Bytes()
	if err != nil {
		err = fmt.Errorf("could not convert NodeID to bytes: %w", err)
		return
	}
	copy(iID[:], idBytes)
	return
}
