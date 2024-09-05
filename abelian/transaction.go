package abelian

import (
	"fmt"
	"github.com/pqabelian/abec/abecryptox/abecryptoxkey"
	api "github.com/pqabelian/abec/sdkapi/v2"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
	"sort"
)

type TxInDesc struct {
	BlockHeight      int64
	BlockID          string
	TxVersion        uint32
	TxID             string
	TxOutIndex       uint8
	TxOutData        []byte
	CoinValue        int64
	CoinSerialNumber []byte
}

func SortTxInDescs(txIndescs []*TxInDesc) error {
	// check firstly
	for i := 0; i < len(txIndescs); i++ {
		_, err := crypto.GetTxoPrivacyLevel(txIndescs[i].TxVersion, txIndescs[i].TxOutData)
		if err != nil {
			sdkLog.Errorf("SortTxInDescs: %v", err)
			return err
		}
	}

	// sort
	sort.SliceStable(txIndescs, func(i, j int) bool {
		coinAddressIPrivacyLevel, _ := crypto.GetTxoPrivacyLevel(txIndescs[i].TxVersion, txIndescs[i].TxOutData)
		coinAddressJPrivacyLevel, _ := crypto.GetTxoPrivacyLevel(txIndescs[j].TxVersion, txIndescs[j].TxOutData)
		if coinAddressIPrivacyLevel != abecryptoxkey.PrivacyLevelPSEUDONYM && coinAddressJPrivacyLevel == abecryptoxkey.PrivacyLevelPSEUDONYM {
			return true
		}
		return false
	})
	return nil
}

type TxOutDesc struct {
	AbelAddress *AbelAddress
	CoinValue   int64
}

func SortTxOutDesc(txOutdescs []*TxOutDesc) error {
	sort.SliceStable(txOutdescs, func(i, j int) bool {
		if txOutdescs[i].AbelAddress.GetCryptoAddress().GetPrivacyLevel() != abecryptoxkey.PrivacyLevelPSEUDONYM &&
			txOutdescs[j].AbelAddress.GetCryptoAddress().GetPrivacyLevel() == abecryptoxkey.PrivacyLevelPSEUDONYM {
			return true
		}
		return false
	})
	return nil
}

type TxDesc struct {
	TxInDescs        []*TxInDesc
	TxOutDescs       []*TxOutDesc
	TxFee            int64
	TxMemo           []byte
	TxRingBlockDescs map[int64]*TxBlockDesc
}

func NewTxDesc(txInDescs []*TxInDesc, txOutDescs []*TxOutDesc, txFee int64, txRingBlockDescs map[int64]*TxBlockDesc) *TxDesc {
	return &TxDesc{
		TxInDescs:        txInDescs,
		TxOutDescs:       txOutDescs,
		TxFee:            txFee,
		TxRingBlockDescs: txRingBlockDescs,
	}
}

type UnsignedRawTx struct {
	Data []byte
}

type TxBlockDesc struct {
	BinData []byte
	Height  int64
}

func NewTxBlockDesc(binData []byte, height int64) *TxBlockDesc {
	return &TxBlockDesc{
		BinData: binData,
		Height:  height,
	}
}

type SignedRawTx struct {
	Data []byte
	TxID string
}

// GenerateUnsignedRawTx make an unsigned transaction,
// which can be signed with singer key by calling GenerateSignedRawTx
func GenerateUnsignedRawTx(txDesc *TxDesc) (*UnsignedRawTx, error) {
	// Prepare outPointsToSpend.
	outPointsToSpend := make([]*api.OutPoint, 0, len(txDesc.TxInDescs))
	for i := 0; i < len(txDesc.TxInDescs); i++ {
		outPoint, err := api.NewOutPointFromTxIdStr(txDesc.TxInDescs[i].TxID, txDesc.TxInDescs[i].TxOutIndex)
		if err != nil {
			sdkLog.Errorf("fail to new outpoint in GenerateUnsignedRawTx with error %v", err)
			return nil, err
		}
		outPointsToSpend = append(outPointsToSpend, outPoint)
	}

	// Prepare serializedBlocksForRingGroup.
	serializedBlocksForRingGroup := getSerializedBlocksForRingGroup(txDesc.TxRingBlockDescs)

	// Prepare txRequestOutputDesc.
	txRequestOutputDescs := make([]*api.TxRequestOutputDesc, 0, len(txDesc.TxOutDescs))
	for i := 0; i < len(txDesc.TxOutDescs); i++ {
		cryptoAddressData := txDesc.TxOutDescs[i].AbelAddress.GetCryptoAddress().Data()
		coinValue := uint64(txDesc.TxOutDescs[i].CoinValue)
		txRequestOutputDesc := api.NewTxRequestOutputDesc(cryptoAddressData, coinValue)
		txRequestOutputDescs = append(txRequestOutputDescs, txRequestOutputDesc)
	}

	// Call API to build the serializedTxRequestDesc.
	serializedTxRequestDesc, err := api.BuildTransferTxRequestDescFromBlocks(
		outPointsToSpend,
		serializedBlocksForRingGroup,
		txRequestOutputDescs,
		uint64(txDesc.TxFee),
		txDesc.TxMemo,
	)
	if err != nil {
		sdkLog.Errorf("fail to build tranasction request from blocks: %v", err)
		return nil, err
	}

	return &UnsignedRawTx{
		Data: serializedTxRequestDesc,
	}, nil
}

// GenerateSignedRawTx signs the unsigned transaction using specified account
func GenerateSignedRawTx(unsignedRawTx *UnsignedRawTx, signerAccounts []Account) (*SignedRawTx, error) {
	if len(signerAccounts) == 0 {
		return nil, fmt.Errorf("no singer specified")
	}
	firstAccountType := signerAccounts[0].AccountType()
	for i := 1; i < len(signerAccounts); i++ {
		if signerAccounts[i].AccountType() != firstAccountType {
			sdkLog.Errorf("all specified account must be the same type")
			return nil, fmt.Errorf("all specified account must be the same type")
		}
	}

	var serializedTxFull []byte
	var txid *api.TxId
	var err error
	switch firstAccountType {
	case AccountTypeSeeds:
		seeds := make([]*api.CryptoRootSeed, 0, len(signerAccounts))
		for i := 0; i < len(signerAccounts); i++ {
			coinSerialNumberKeyMaterial, coinValueKeyMaterial, coinDetectorKeyMaterial := signerAccounts[i].ViewKeyMaterial()
			coinSpendSecretKeyMaterial := signerAccounts[i].SpendKeyMaterial()
			signerViewAccount := signerAccounts[i].(*RootSeedAccount)
			seeds = append(seeds, api.NewRootSeed(
				signerViewAccount.cryptoScheme,
				signerViewAccount.privacyLevel,
				coinSpendSecretKeyMaterial,
				coinSerialNumberKeyMaterial,
				coinValueKeyMaterial,
				coinDetectorKeyMaterial,
			))
		}
		serializedTxFull, txid, err = api.CreateTransferTxByRootSeed(unsignedRawTx.Data, seeds)
		if err != nil {
			sdkLog.Errorf("fail to create transfer tx by root seed: %v", err)
			return nil, err
		}
	case AccountTypeKeys:
		// Prepare cryptoKeys.
		cryptoKeys := make([]*api.CryptoKey, 0, len(signerAccounts))
		for i := 0; i < len(signerAccounts); i++ {
			coinSerialNumberKeyMaterial, coinValueKeyMaterial, coinDetectorKeyMaterial := signerAccounts[i].ViewKeyMaterial()
			coinSpendSecretKeyMaterial := signerAccounts[i].SpendKeyMaterial()
			signerViewAccount := signerAccounts[i].(*CryptoKeysAccount)
			cryptoKeys = append(cryptoKeys, api.NewCryptoKey(
				signerViewAccount.cryptoAddress.Data(),
				coinSpendSecretKeyMaterial,
				coinSerialNumberKeyMaterial,
				coinValueKeyMaterial,
				coinDetectorKeyMaterial,
			))
		}

		// Call API to create the signed raw tx.
		serializedTxFull, txid, err = api.CreateTransferTxByCryptoKeys(unsignedRawTx.Data, cryptoKeys)
		if err != nil {
			sdkLog.Errorf("fail to create transfer tx by crypto keys: %v", err)
			return nil, err
		}
	default:
		return nil, ErrInvalidAccountType
	}

	return &SignedRawTx{
		Data: serializedTxFull,
		TxID: txid.String(),
	}, nil
}
func getSerializedBlocksForRingGroup(ringBlockDescs map[int64]*TxBlockDesc) [][]byte {
	heights := make([]int64, 0, len(ringBlockDescs))
	for height := range ringBlockDescs {
		heights = append(heights, height)
	}

	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	serializedBlocksForRingGroup := make([][]byte, 0, len(ringBlockDescs))
	for i := 0; i < len(heights); i++ {
		serializedBlocksForRingGroup = append(serializedBlocksForRingGroup, ringBlockDescs[heights[i]].BinData)
	}

	return serializedBlocksForRingGroup
}
