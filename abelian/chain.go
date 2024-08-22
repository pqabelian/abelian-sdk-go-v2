package abelian

import (
	"fmt"
	api "github.com/pqabelian/abec/sdkapi/v2"
)

type NetworkID uint8

const (
	MainNet       NetworkID = 0
	RegressionNet NetworkID = 1
	TestNet       NetworkID = 2
	SimNet        NetworkID = 3
)

func (n NetworkID) String() string {
	switch n {
	case MainNet:
		return "mainnet"
	case RegressionNet:
		return "regressionnet"
	case TestNet:
		return "testnet"
	case SimNet:
		return "simnet"
	default:
		return "unknown"
	}
}

var NetName2NetID = map[string]NetworkID{
	"mainnet":       MainNet,
	"regressionnet": RegressionNet,
	"testnet":       TestNet,
	"simnet":        SimNet,
}

func GetTxoRingSizeByBlockHeight(height int32) uint8 {
	return api.GetTxoRingSizeByBlockHeight(height)
}

func GetBlockNumPerRingGroupByBlockHeight(height int32) uint8 {
	return api.GetBlockNumPerRingGroupByBlockHeight(height)
}

func GetCoinbaseMaturity() int32 {
	return 200
}

func GetRingBlockHeights(height int32) []int32 {
	blockNumPerGroup := GetBlockNumPerRingGroupByBlockHeight(height)
	firstRingBlockHeight := height - height%int32(blockNumPerGroup)

	ringBlockHeights := make([]int32, 0, blockNumPerGroup)
	for i := int32(0); i < int32(blockNumPerGroup); i++ {
		ringBlockHeights = append(ringBlockHeights, firstRingBlockHeight+i)
	}
	return ringBlockHeights
}

func GetRingBlockGroupByHeight(client *Client, height int32) ([][]byte, error) {
	blockNumPerGroup := GetBlockNumPerRingGroupByBlockHeight(height)
	firstRingBlockHeight := height - height%int32(blockNumPerGroup)

	serializedBlockGroups := make([][]byte, 0, blockNumPerGroup)
	for i := int32(0); i < int32(blockNumPerGroup); i++ {
		blockBytes, err := client.GetBlockBytesByHeight(firstRingBlockHeight + i)
		if err != nil {
			return nil, fmt.Errorf("fail to get block group: %v", err)
		}
		serializedBlockGroups = append(serializedBlockGroups, blockBytes)
	}
	return serializedBlockGroups, nil
}

func EstimateTxFee(txinDescs []*TxInDesc, txOutDescs []*TxOutDesc) int64 {
	return 1_000_000
}

func NeutrinoToAbel(neutrinoAmount int64) float64 {
	return float64(neutrinoAmount) / 1e7
}

func AbelToNeutrino(abelAmount float64) int64 {
	return int64(abelAmount * 1e7)
}
