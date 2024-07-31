package abelian

import (
	"bytes"
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/chain"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

type AccountPrivacyLevel int

const (
	AccountPrivacyLevelFullPrivacyOld AccountPrivacyLevel = 0
	AccountPrivacyLevelFullPrivacy    AccountPrivacyLevel = 1
	AccountPrivacyLevelPseudonym      AccountPrivacyLevel = 2
)

func NewAccount(accountPrivacyLevel AccountPrivacyLevel) (Account, []byte, error) {
	var cryptoScheme crypto.CryptoScheme
	var privacyLevel crypto.PrivacyLevel
	switch accountPrivacyLevel {
	case AccountPrivacyLevelFullPrivacyOld:
		cryptoScheme = crypto.CryptoSchemePQRingCT
		privacyLevel = crypto.PrivacyLevelFullPrivacyPre
		randSeeds, err := crypto.GenerateSeed(cryptoScheme, privacyLevel)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to genereate seed for account")
		}
		keysAndAddressByRandSeeds, err := crypto.GenerateCryptoKeysAndAddressByRandSeeds(randSeeds)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to genereate keys and address")
		}
		account := NewCryptoKeysAccount(
			keysAndAddressByRandSeeds.SpendSecretKey,
			keysAndAddressByRandSeeds.SerialNoSecretKey,
			keysAndAddressByRandSeeds.ViewSecretKey,
			keysAndAddressByRandSeeds.DetectorKey,
			keysAndAddressByRandSeeds.CryptoAddress,
		)
		seedBytes, err := randSeeds.Serialize()
		return account, seedBytes, err
	case AccountPrivacyLevelFullPrivacy:
		cryptoScheme = crypto.CryptoSchemePQRingCTX
		privacyLevel = crypto.PrivacyLevelFullPrivacyRand

	case AccountPrivacyLevelPseudonym:
		cryptoScheme = crypto.CryptoSchemePQRingCTX
		privacyLevel = crypto.PrivacyLevelPseudonym
	default:
		return nil, nil, fmt.Errorf("invalid privacy level for account")
	}
	rootSeeds, err := crypto.GenerateSeed(cryptoScheme, privacyLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to genereate seed for account")
	}

	account := NewRootSeedAccount(
		cryptoScheme,
		privacyLevel,
		rootSeeds.CoinSpendKeySeed(),
		rootSeeds.CoinSerialNumberKeySeed(),
		rootSeeds.CoinValueKeySeed(),
		rootSeeds.CoinDetectorKey(),
	)
	seedBytes, err := rootSeeds.Serialize()
	return account, seedBytes, err
}

type AccountType int

const (
	AccountTypeSeeds AccountType = iota
	AccountTypeKeys
)

// ViewAccount encapsulates the ability to
// - determine whether the coin belongs to the corresponding account
// - generate serial number for specified coins
type ViewAccount interface {
	ReceiveCoin(txVersion uint32, txOutData []byte) (success bool, v uint64, err error)
	GenerateSerialNumbersWithBlocks(coinIDs []*chain.CoinID, serializedBlocksForRingGroup [][]byte) (coinSerialNumbers [][]byte, err error)
	ViewKeyMaterial() ([]byte, []byte, []byte)

	AccountType() AccountType
}

var _ ViewAccount = &RootSeedViewAccount{}
var _ ViewAccount = &CryptoKeysViewAccount{}

type RootSeedViewAccount struct {
	cryptoScheme            crypto.CryptoScheme
	privacyLevel            crypto.PrivacyLevel
	coinSerialNumberKeySeed []byte
	coinValueKeySeed        []byte
	coinDetectorKey         []byte
}

func NewRootSeedViewAccount(
	cryptoScheme crypto.CryptoScheme,
	privacyLevel crypto.PrivacyLevel,
	coinSerialNumberKeySeed []byte,
	coinValueKeySeed []byte,
	coinDetectorKey []byte,
) *RootSeedViewAccount {
	return &RootSeedViewAccount{cryptoScheme: cryptoScheme, privacyLevel: privacyLevel, coinSerialNumberKeySeed: coinSerialNumberKeySeed, coinValueKeySeed: coinValueKeySeed, coinDetectorKey: coinDetectorKey}
}

func (account *RootSeedViewAccount) AccountType() AccountType {
	return AccountTypeSeeds
}
func (account *RootSeedViewAccount) GenerateSerialNumbersWithBlocks(coinIDs []*chain.CoinID, serializedBlocksForRingGroup [][]byte) ([][]byte, error) {
	if len(coinIDs) == 0 {
		return nil, nil
	}
	// Prepare outPoints.
	outPoints := make([]*crypto.OutPoint, len(coinIDs))
	for i := 0; i < len(coinIDs); i++ {
		outPoint, err := crypto.NewOutPointFromTxId(coinIDs[i].TxID, coinIDs[i].Index)
		if err != nil {
			return nil, err
		}
		outPoints[i] = outPoint
	}

	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByRootSeeds(outPoints, serializedBlocksForRingGroup, account.coinSerialNumberKeySeed)
	if err != nil {
		return nil, err
	}

	// Convert serial numbers to []byte] type and return them.
	coinSerialNumbers := make([][]byte, len(coinIDs))
	for i := 0; i < len(serialNumbers); i++ {
		coinSerialNumbers[i] = serialNumbers[i]
	}

	return coinSerialNumbers, nil
}
func (account *RootSeedViewAccount) ReceiveCoin(txVersion uint32, txOutData []byte) (success bool, v uint64, err error) {
	privacyLevel, err := crypto.GetTxoPrivacyLevel(txVersion, txOutData)
	if err != nil {
		return false, 0, err
	}
	if account.privacyLevel != privacyLevel {
		return false, 0, nil
	}

	success, err = crypto.TxoCoinDetectByCoinDetectorRootKey(txVersion, txOutData, account.coinDetectorKey)
	if err != nil {
		return false, 0, err
	}
	if !success {
		return false, 0, nil
	}

	success, v, err = crypto.TxoCoinReceiveByRootSeeds(txVersion, txOutData, account.coinValueKeySeed, account.coinDetectorKey)
	if err != nil {
		return false, 0, err
	}
	return success, v, nil
}
func (account *RootSeedViewAccount) ViewKeyMaterial() ([]byte, []byte, []byte) {
	var coinDetectorKey []byte
	if account.coinDetectorKey != nil {
		coinDetectorKey = account.coinDetectorKey
	}
	return account.coinSerialNumberKeySeed, account.coinValueKeySeed, coinDetectorKey
}

type CryptoKeysViewAccount struct {
	cryptoScheme      crypto.CryptoScheme
	privacyLevel      crypto.PrivacyLevel
	serialNoSecretKey []byte
	viewSecretKey     []byte
	detectorKey       []byte
	cryptoAddress     *crypto.CryptoAddress
}

func NewCryptoKeyViewAccount(cryptoScheme crypto.CryptoScheme, privacyLevel crypto.PrivacyLevel,
	serialNoSecretKey []byte, viewSecretKey []byte,
	detectorKey []byte, cryptoAddress *crypto.CryptoAddress) *CryptoKeysViewAccount {

	return &CryptoKeysViewAccount{
		cryptoScheme:      cryptoScheme,
		privacyLevel:      privacyLevel,
		serialNoSecretKey: serialNoSecretKey,
		viewSecretKey:     viewSecretKey,
		detectorKey:       detectorKey,
		cryptoAddress:     cryptoAddress,
	}
}
func (account *CryptoKeysViewAccount) AccountType() AccountType {
	return AccountTypeKeys
}
func (account *CryptoKeysViewAccount) GenerateSerialNumbersWithBlocks(coinIDs []*chain.CoinID, serializedBlocksForRingGroup [][]byte) ([][]byte, error) {
	if len(coinIDs) == 0 {
		return nil, nil
	}
	// Prepare outPoints.
	outPoints := make([]*crypto.OutPoint, len(coinIDs))
	for i := 0; i < len(coinIDs); i++ {
		outPoint, err := crypto.NewOutPointFromTxId(coinIDs[i].TxID, coinIDs[i].Index)
		if err != nil {
			return nil, err
		}
		outPoints[i] = outPoint
	}

	// Prepare cryptoSecretKeys.
	cryptoSerialNumberSecretKeys := make([][]byte, len(coinIDs))
	for i := 0; i < len(coinIDs); i++ {
		cryptoSerialNumberSecretKeys[i] = account.serialNoSecretKey
	}

	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByKeys(outPoints, serializedBlocksForRingGroup, cryptoSerialNumberSecretKeys)
	if err != nil {
		return nil, err
	}

	// Convert serial numbers to []byte] type and return them.
	coinSerialNumbers := make([][]byte, len(coinIDs))
	for i := 0; i < len(serialNumbers); i++ {
		coinSerialNumbers[i] = serialNumbers[i]
	}

	return coinSerialNumbers, nil
}

func (account *CryptoKeysViewAccount) ReceiveCoin(txVersion uint32, txOutData []byte) (success bool, v uint64, err error) {
	coinAddressFromSerializedTxOut, err := crypto.DecodeCoinAddressFromSerializedTxOutData(txVersion, txOutData)
	if err != nil {
		return false, 0, err
	}
	coinAddressFromCryptoAddress := account.cryptoAddress.GetCoinAddress()
	if !bytes.Equal(coinAddressFromCryptoAddress.Data(), coinAddressFromSerializedTxOut.Data()) {
		return false, 0, nil
	}

	copiedVsk := make([]byte, len(account.viewSecretKey))
	copy(copiedVsk, account.viewSecretKey)
	success, v, err = crypto.TxoCoinReceiveByKeys(txVersion, txOutData, account.cryptoAddress.Data(), copiedVsk)
	if err != nil {
		return false, 0, err
	}
	return success, v, nil
}
func (account *CryptoKeysViewAccount) ViewKeyMaterial() ([]byte, []byte, []byte) {
	var coinSerialNoSecretKey []byte
	if account.serialNoSecretKey != nil {
		coinSerialNoSecretKey = account.serialNoSecretKey
	}
	var coinViewSecretKey []byte
	if account.viewSecretKey != nil {
		coinViewSecretKey = account.viewSecretKey
	}

	var coinDetectorKey []byte
	if account.detectorKey != nil {
		coinDetectorKey = account.detectorKey
	}
	return coinSerialNoSecretKey, coinViewSecretKey, coinDetectorKey
}

type Account interface {
	ViewAccount
	SpendKeyMaterial() []byte
	Dump() ([]byte, error)
	ViewAccount() ViewAccount
}

var _ Account = &RootSeedAccount{}
var _ Account = &CryptoKeysAccount{}

type RootSeedAccount struct {
	RootSeedViewAccount
	coinSpendKeySeed []byte
}

func (account *RootSeedAccount) SpendKeyMaterial() []byte {
	return account.coinSpendKeySeed
}
func (account *RootSeedAccount) ViewAccount() ViewAccount {
	return &account.RootSeedViewAccount
}
func (account *RootSeedAccount) Dump() ([]byte, error) {
	seeds, _ := crypto.NewRootSeeds(
		account.cryptoScheme,
		account.privacyLevel,
		account.coinSpendKeySeed,
		account.coinSerialNumberKeySeed,
		account.coinValueKeySeed,
		account.coinDetectorKey,
	)
	return seeds.Serialize()
}

func NewRootSeedAccount(cryptoScheme crypto.CryptoScheme, privacyLevel crypto.PrivacyLevel,
	coinSpendKeySeed []byte, coinSerialNumberKeySeed []byte, coinValueKeySeed []byte,
	coinDetectorKey []byte) *RootSeedAccount {
	return &RootSeedAccount{
		RootSeedViewAccount: RootSeedViewAccount{
			cryptoScheme:            cryptoScheme,
			privacyLevel:            privacyLevel,
			coinSerialNumberKeySeed: coinSerialNumberKeySeed,
			coinValueKeySeed:        coinValueKeySeed,
			coinDetectorKey:         coinDetectorKey,
		},
		coinSpendKeySeed: coinSpendKeySeed,
	}
}
func NewRootSeedAccountFromViewAccount(viewAccount RootSeedViewAccount, coinSpendKeySeed []byte) *RootSeedAccount {
	return &RootSeedAccount{
		RootSeedViewAccount: viewAccount,
		coinSpendKeySeed:    coinSpendKeySeed,
	}
}

type CryptoKeysAccount struct {
	CryptoKeysViewAccount
	spendSecretKey []byte
}

func (account *CryptoKeysAccount) SpendKeyMaterial() []byte {
	return account.spendSecretKey
}
func (account *CryptoKeysAccount) ViewAccount() ViewAccount {
	return &account.CryptoKeysViewAccount
}
func (account *CryptoKeysAccount) Dump() ([]byte, error) {
	seeds, _ := crypto.NewRandSeeds(
		account.cryptoScheme,
		account.privacyLevel,
		account.spendSecretKey,
		account.serialNoSecretKey,
		account.viewSecretKey,
		account.detectorKey,
		nil,
	)
	return seeds.Serialize()
}
func NewCryptoKeysAccount(spendSecretKey []byte, serialNoSecretKey []byte, viewSecretKey []byte, detectorKey []byte, cryptoAddress *crypto.CryptoAddress) *CryptoKeysAccount {
	return &CryptoKeysAccount{
		CryptoKeysViewAccount: CryptoKeysViewAccount{
			serialNoSecretKey: serialNoSecretKey,
			viewSecretKey:     viewSecretKey,
			detectorKey:       detectorKey,
			cryptoAddress:     cryptoAddress,
		},
		spendSecretKey: spendSecretKey,
	}
}
func NewCryptoKeysAccountFromViewAccount(viewAccount CryptoKeysViewAccount, spendSecretKey []byte) *CryptoKeysAccount {
	return &CryptoKeysAccount{
		CryptoKeysViewAccount: viewAccount,
		spendSecretKey:        spendSecretKey,
	}
}
