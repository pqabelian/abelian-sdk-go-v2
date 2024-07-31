package abelian

import "github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"

type TxInDesc struct {
	Height           int64
	BlockID          string
	TxVersion        uint32
	TxID             string
	TxOutIndex       uint8
	TxOutData        []byte
	CoinValue        int64
	CoinSerialNumber []byte
}
type TxOutDesc struct {
	AbelAddress *crypto.AbelAddress
	CoinValue   int64
}

type TxDesc struct {
	TxInDescs        []*TxInDesc
	TxOutDescs       []*TxOutDesc
	TxFee            int64
	TxMemo           []byte
	TxRingBlockDescs map[int64][]byte
}

type UnsignedRawTx struct {
	Data []byte
}

type SignedRawTx struct {
	Data []byte
	TxID string
}
