package chain

type NetworkID uint8

const (
	MainNet       NetworkID = 0
	RegressionNet NetworkID = 1
	TestNet3      NetworkID = 2
	SimNet        NetworkID = 3
)

var NetName2NetID = map[string]NetworkID{
	"mainnet":       MainNet,
	"regressionnet": RegressionNet,
	"testnet3":      TestNet3,
	"simnet":        SimNet,
}
