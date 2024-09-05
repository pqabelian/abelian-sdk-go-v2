package common

import (
	"encoding/json"
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"os"
)

type Config struct {
	NetworkID  abelian.NetworkID `json:"network_id"`
	DBFileName string            `json:"db_file_name"`
	RPC        struct {
		Endpoint string `json:"endpoint"`
		UserName string `json:"username"`
		Password string `json:"password"`
	} `json:"rpc"`
	CoinScan struct {
		StartHeight int64 `json:"start_height"`
		EndHeight   int64 `json:"end_height"`
	} `json:"coin_scan"`
	TxTrack struct {
		StartHeight int64 `json:"start_height"`
		EndHeight   int64 `json:"end_height"`
	} `json:"tx_track"`
}

var config Config
var Client *abelian.Client

func init() {
	configFile, err := os.Open("abelian-sdk-conf.json")
	if err != nil {
		panic(err)
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	clientConfig := abelian.NewClientConfig(
		config.RPC.Endpoint,
		abelian.WithAuth(config.RPC.UserName, config.RPC.Password),
	)
	Client, err = abelian.NewClient(clientConfig)

	// Assert network
	info, err := Client.GetChainInfo()
	if err != nil {
		panic(fmt.Errorf("fail to get block hash: %v", err))
	}
	fmt.Printf("chain info: %#+v\n", info)

	// assert network id
	if info.NetID == uint8(config.NetworkID) {
		fmt.Printf("network id is matched: %s\n", config.NetworkID)
	} else {
		panic(fmt.Errorf("network id is unmatched: %d", info.NetID))
	}
}

func GetNetworkID() abelian.NetworkID {
	return config.NetworkID
}
func GetDBFileName() string {
	return config.DBFileName
}
func GetCoinScanStartHeight() int64 {
	return config.CoinScan.StartHeight
}
func GetCoinScanEndHeight() int64 {
	return config.CoinScan.EndHeight
}
func GetTxTrackStartHeight() int64 {
	return config.TxTrack.StartHeight
}
func GetTxTrackEndHeight() int64 {
	return config.TxTrack.EndHeight
}
