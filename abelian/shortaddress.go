package abelian

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/chain"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

const (
	SHORT_ABEL_ADDRESS_LENGTH = 66
)

// ShortAbelAddress define short address
type ShortAbelAddress struct {
	data []byte
}

func (address *ShortAbelAddress) Data() []byte {
	return address.data
}

func (address *ShortAbelAddress) Validate() error {
	if len(address.data) != SHORT_ABEL_ADDRESS_LENGTH {
		return fmt.Errorf("short abel address data length is not %d", SHORT_ABEL_ADDRESS_LENGTH)
	}

	if address.data[0] != 0xab {
		return fmt.Errorf("short abel address data is not prefixed with 0xab")
	}

	chainID := address.data[1] - 0xe1
	if chainID < 0 || chainID > 15 {
		return fmt.Errorf("short abel address chain id is not in range [0, 15]")
	}

	return nil
}

// NewShortAbelAddress
/* TODO confirm the short abel address */
func NewShortAbelAddress(chainID chain.NetworkID, fingerprint []byte, cryptoAddressHash []byte) (*ShortAbelAddress, error) {
	saData := make([]byte, 0, 2+len(fingerprint)+len(cryptoAddressHash))
	saData = append(saData, 0xab, 0xe1+byte(chainID))
	saData = append(saData, fingerprint...)
	saData = append(saData, cryptoAddressHash...)

	if len(saData) != SHORT_ABEL_ADDRESS_LENGTH {
		return nil, crypto.ErrInvalidAddress
	}

	return &ShortAbelAddress{data: saData}, nil
}

func GetShortAbelAddressFromAbelAddress(address *chain.AbelAddress) *ShortAbelAddress {
	shortAbelAddress, _ := NewShortAbelAddress(
		address.GetNetID(),
		address.GetCryptoAddress().GetCoinAddress().Fingerprint(),
		nil, /* TODO confirm the short abel address */
	)
	return shortAbelAddress
}
