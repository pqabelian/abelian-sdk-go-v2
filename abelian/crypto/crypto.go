package crypto

import (
	"fmt"
	api "github.com/pqabelian/abec/sdkapi/v2"
)

type CryptoScheme = api.CryptoScheme

const (
	CryptoSchemePQRingCT  = api.CryptoSchemePQRingCT
	CryptoSchemePQRingCTX = api.CryptoSchemePQRingCTX
)

func CryptoSchemeSerializeSize() int {
	return api.CryptoSchemeSerializeSize()
}
func SerializeCryptoScheme(cryptoScheme CryptoScheme) []byte {
	return api.SerializeCryptoScheme(cryptoScheme)
}
func DeserializeCryptoScheme(serializedCryptoScheme []byte) (CryptoScheme, error) {
	return api.DeserializeCryptoScheme(serializedCryptoScheme)
}

// PrivacyLevel alias type for api.PrivacyLevel to identifier different privacy level
// Defined by a cryptographic layer to differentiate for different target
//
// currently, two full-privacy addresses and one pseudonym address are supported
type PrivacyLevel = api.PrivacyLevel

const (
	PrivacyLevelFullPrivacyPre  PrivacyLevel = api.PrivacyLevelRINGCTPre // for back-compatibility
	PrivacyLevelFullPrivacyRand              = api.PrivacyLevelRINGCT
	PrivacyLevelPseudonym                    = api.PrivacyLevelPSEUDONYM
)

func GetCryptoSchemeParamSeedBytesLen(cryptoScheme CryptoScheme) (int, error) {
	return api.GetCryptoSchemeParamSeedBytesLen(cryptoScheme)
}
func GetParamKeyGenPublicRandBytesLen(cryptoScheme CryptoScheme) (int, error) {
	return api.GetParamKeyGenPublicRandBytesLen(cryptoScheme)
}

// GenerateSeed generate a safe seed
// For back-compatibility, hoisting origin implementation here
//
// for CryptoSchemePQRingCT: to keep back-compatibility
//  1. PrivacyLevelFullPrivacyPre
//     cryptoScheme || coinSpendKeyRootSeed || coinValueKeyRootSeed
//
// for CryptoSchemePQRingCTX:
//  1. PrivacyLevelFullPrivacyRand
//     cryptoScheme || privacyLevel || coinSpendKeyRootSeed || coinSerialNumberKeyRootSeed || coinValueKeyRootSeed || coinDetectorRootKey
//  2. PrivacyLevelPseudonym
//     cryptoScheme || privacyLevel || coinSpendKeyRootSeed || coinDetectorRootKey
func GenerateSeed(cryptoScheme CryptoScheme, privacyLevel PrivacyLevel) (*CryptoSeeds, error) {
	underlyingSeedLen, err := api.GetCryptoSchemeParamSeedBytesLen(cryptoScheme)
	if err != nil {
		return nil, err
	}
	switch cryptoScheme {
	case CryptoSchemePQRingCT:
		if privacyLevel != PrivacyLevelFullPrivacyPre {
			return nil, fmt.Errorf("invalid privacy level %d for crypto scheme %d", privacyLevel, cryptoScheme)
		}
		coinAddressKeySeed := RandomBytes(underlyingSeedLen)
		coinValueKeySeed := RandomBytes(underlyingSeedLen)
		return NewRandSeeds(cryptoScheme, privacyLevel, coinAddressKeySeed, nil, coinValueKeySeed, nil, nil)
	case CryptoSchemePQRingCTX:
		if privacyLevel != PrivacyLevelFullPrivacyRand && privacyLevel != PrivacyLevelPseudonym {
			return nil, fmt.Errorf("invalid privacy level %d for crypto scheme %d", privacyLevel, cryptoScheme)
		}
		coinSpendKeyRootSeed := RandomBytes(underlyingSeedLen)
		var coinSerialNumberKeyRootSeed, coinValueKeyRootSeed []byte
		if privacyLevel == PrivacyLevelFullPrivacyRand {
			coinSerialNumberKeyRootSeed = RandomBytes(underlyingSeedLen)
			coinValueKeyRootSeed = RandomBytes(underlyingSeedLen)
		}
		coinDetectorRootKey := RandomBytes(underlyingSeedLen)
		return NewRootSeeds(cryptoScheme, privacyLevel, coinSpendKeyRootSeed, coinSerialNumberKeyRootSeed, coinValueKeyRootSeed, coinDetectorRootKey)

	default:
		return nil, fmt.Errorf("%v, got %v", ErrInvalidCryptoScheme, cryptoScheme)
	}
}

// GenerateCryptoKeysAndAddressBySeedBytes generate an address key pair from specified seeds
// NOTE: when seedsType is root
func GenerateCryptoKeysAndAddressBySeedBytes(seedBytes []byte) (*CryptoKeysAndAddress, error) {
	cryptoSeeds, err := deserializeSeed(seedBytes)
	if err != nil {
		return nil, err
	}
	if cryptoSeeds.seedsType == seedsTypeRoot {
		return GenerateCryptoKeysAndAddressByRootSeeds(cryptoSeeds)
	} else if cryptoSeeds.seedsType == seedsTypeRand {
		return GenerateCryptoKeysAndAddressByRandSeeds(cryptoSeeds)
	} else {
		return nil, AssertError("call GenerateCryptoKeysAndAddressBySeedBytes with invalid seeds")
	}
}

// GenerateCryptoKeysAndAddressByRootSeeds generate an address key pair from root seeds
// NOTE:
// 1. Multiple call use will produce DIFFERENT pairs
// 2. SHOULD call ONLY when crypto scheme is CryptoSchemePQRingCTX
//
// Two ways to generate the same address pair:
// 1. use ExtractPublicRandFromCryptoAddress to extract the public rand from the generated address,
// and then call GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand to generate
// 2. use ExtractPublicRandFromCryptoAddress to extract the public rand from the generated address,
// and with that call GenerateRandSeedsByRootSeedsFromPublicRand to generate the rand seed,
// and then call GenerateCryptoKeysAndAddressByRandSeeds to generate
func GenerateCryptoKeysAndAddressByRootSeeds(rootSeeds *CryptoSeeds) (*CryptoKeysAndAddress, error) {
	if rootSeeds.seedsType != seedsTypeRoot {
		return nil, AssertError("call GenerateCryptoKeysAndAddressByRootSeeds with invalid type seeds")
	}
	cryptoAddressBytes, cryptoSpendSecretKey,
		cryptoSerialNoSecretKey, cryptoViewSecretKey, cryptoDetectorKey, err := api.CryptoAddressKeyGenByRootSeeds(
		rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		rootSeeds.coinSpendKeySeed, rootSeeds.coinSerialNumberKeySeed,
		rootSeeds.coinValueKeySeed, rootSeeds.coinDetectorKey)
	if err != nil {
		return nil, err
	}

	cryptoAddress, err := NewCryptoAddress(cryptoAddressBytes)
	if err != nil {
		return nil, err
	}

	cryptoKeysAndAddress := &CryptoKeysAndAddress{
		SpendSecretKey:    cryptoSpendSecretKey,
		SerialNoSecretKey: cryptoSerialNoSecretKey,
		ViewSecretKey:     cryptoViewSecretKey,
		DetectorKey:       cryptoDetectorKey,
		CryptoAddress:     cryptoAddress,
	}

	return cryptoKeysAndAddress, nil
}

// ExtractPublicRandFromCryptoAddress extract public rand from crypto address
func ExtractPublicRandFromCryptoAddress(cryptoAddress *CryptoAddress) ([]byte, error) {
	publicRand, err := api.ExtractPublicRandFromCryptoAddress(cryptoAddress.Data())
	return publicRand, err
}

// GenerateRandSeedsByRootSeedsFromPublicRand generate rand seed with specified root seed and public rand
// NOTE: SHOULD call ONLY when crypto scheme is CryptoSchemePQRingCTX
func GenerateRandSeedsByRootSeedsFromPublicRand(rootSeedBytes []byte, publicRand []byte) (*CryptoSeeds, error) {
	rootSeeds, err := deserializeSeed(rootSeedBytes)
	if err != nil {
		return nil, err
	}
	if rootSeeds.seedsType != seedsTypeRoot {
		return nil, AssertError("call GenerateRandSeedsByRootSeedsFromPublicRand with invalid seeds")
	}

	if rootSeeds.cryptoScheme != CryptoSchemePQRingCTX {
		return nil, fmt.Errorf("expected crypto scheme %d, but got %d ", CryptoSchemePQRingCTX, rootSeeds.cryptoScheme)
	}
	if rootSeeds.privacyLevel != PrivacyLevelFullPrivacyRand && rootSeeds.privacyLevel != PrivacyLevelPseudonym {
		return nil, fmt.Errorf("invalid privacy level %d for crypto scheme %d", rootSeeds.privacyLevel, rootSeeds.cryptoScheme)
	}

	coinSpendKeyRandSeed, coinSerialNumberKeyRandSeed,
		coinValueKeyRandSeed, coinDetectorKey, err := api.RandSeedsGenByRootSeedsFromPublicRand(
		rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		rootSeeds.coinSpendKeySeed, rootSeeds.coinSerialNumberKeySeed,
		rootSeeds.coinValueKeySeed, rootSeeds.coinDetectorKey,
		publicRand)
	if err != nil {
		return nil, fmt.Errorf("fail to generate crypto seed from root seed")
	}

	return NewRandSeeds(rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		coinSpendKeyRandSeed, coinSerialNumberKeyRandSeed,
		coinValueKeyRandSeed, coinDetectorKey,
		publicRand)
}

// GenerateCryptoKeysAndAddressByRandSeeds generate an address key pair from rand seeds
// Different from GenerateCryptoKeysAndAddressByRootSeeds, multiple call use will produce THE SAME pairs
func GenerateCryptoKeysAndAddressByRandSeeds(randSeeds *CryptoSeeds) (*CryptoKeysAndAddress, error) {
	if randSeeds.seedsType != seedsTypeRand {
		return nil, AssertError("call GenerateCryptoKeysAndAddressByRandSeeds invalid seeds")
	}
	var coinDetectorKey []byte
	if randSeeds.coinDetectorKey != nil {
		coinDetectorKey = randSeeds.coinDetectorKey
	}

	cryptoAddressBytes, cryptoSpendSecretKey, cryptoSerialNoSecretKey,
		cryptoViewSecretKey, cryptoDetectorKey, err := api.CryptoAddressKeyGenByRandSeeds(
		randSeeds.cryptoScheme, randSeeds.privacyLevel,
		randSeeds.coinSpendKeySeed, randSeeds.coinSerialNumberKeySeed,
		randSeeds.coinValueKeySeed, coinDetectorKey,
		randSeeds.publicRand)
	if err != nil {
		return nil, err
	}

	cryptoAddress, err := NewCryptoAddress(cryptoAddressBytes)
	if err != nil {
		return nil, err
	}
	cryptoKeysAndAddress := &CryptoKeysAndAddress{
		SpendSecretKey:    cryptoSpendSecretKey,
		SerialNoSecretKey: cryptoSerialNoSecretKey,
		ViewSecretKey:     cryptoViewSecretKey,
		DetectorKey:       cryptoDetectorKey,
		CryptoAddress:     cryptoAddress,
	}

	return cryptoKeysAndAddress, nil
}

// GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand generate the address key pair from root seed
// and public rand which can extracted from crypto address by calling ExtractPublicRandFromCryptoAddress
func GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand(rootSeedBytes []byte, publicRand []byte) (*CryptoKeysAndAddress, error) {
	rootSeeds, err := deserializeSeed(rootSeedBytes)
	if err != nil {
		return nil, err
	}
	if rootSeeds.seedsType != seedsTypeRoot {
		return nil, AssertError("call GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand with invalid seeds")
	}

	cryptoAddressBytes, cryptoSpendSecretKey, cryptoSerialNoSecretKey,
		cryptoViewSecretKey, cryptoDetectorKey, err := api.CryptoAddressKeyReGenByRootSeedsFromPublicRand(
		rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		rootSeeds.coinSpendKeySeed, rootSeeds.coinSerialNumberKeySeed,
		rootSeeds.coinValueKeySeed, rootSeeds.coinDetectorKey,
		publicRand)
	if err != nil {
		return nil, err
	}
	cryptoAddress, err := NewCryptoAddress(cryptoAddressBytes)
	if err != nil {
		return nil, err
	}
	cryptoKeysAndAddress := &CryptoKeysAndAddress{
		SpendSecretKey:    cryptoSpendSecretKey,
		SerialNoSecretKey: cryptoSerialNoSecretKey,
		ViewSecretKey:     cryptoViewSecretKey,
		DetectorKey:       cryptoDetectorKey,
		CryptoAddress:     cryptoAddress,
	}

	return cryptoKeysAndAddress, nil
}

// DecodeCoinAddressFromSerializedTxOutData extract coin address from serialized transaction output
func DecodeCoinAddressFromSerializedTxOutData(txVersion uint32, txOutData []byte) (CoinAddress, error) {
	// potentially use the latest transaction version default
	coinAddressData, err := api.ExtractCoinAddressFromSerializedTxOut(txVersion, txOutData)
	if err != nil {
		return nil, err
	}

	return NewCoinAddress(coinAddressData)
}

type OutPoint = api.OutPoint

func NewOutPointFromTxId(txID string, index uint8) (*OutPoint, error) {
	outPoint, err := api.NewOutPointFromTxIdStr(txID, index)
	return outPoint, err
}

func GetTxoPrivacyLevel(txVersion uint32, txOutData []byte) (PrivacyLevel, error) {
	return api.GetTxoPrivacyLevel(txVersion, txOutData)
}

func TxoCoinDetectByCoinDetectorRootKey(txVersion uint32, serializedTxOut []byte, coinDetectorRootKey []byte) (bool, error) {
	return api.TxoCoinDetectByCoinDetectorRootKey(txVersion, serializedTxOut, coinDetectorRootKey)
}

func TxoCoinReceiveByRootSeeds(txVersion uint32, serializedTxOut []byte, coinValueKeyRootSeed []byte, coinDetectorRootKey []byte) (bool, uint64, error) {
	return api.TxoCoinReceiveByRootSeeds(txVersion, serializedTxOut, coinValueKeyRootSeed, coinDetectorRootKey)
}

func GenerateCoinSerialNumberByRootSeeds(
	outPoints []*OutPoint,
	serializedBlocksForRingGroup [][]byte,
	coinSerialNumberKeyRootSeed []byte) (serialNumbers [][]byte, err error) {
	return api.GenerateCoinSerialNumberByRootSeeds(outPoints, serializedBlocksForRingGroup, coinSerialNumberKeyRootSeed)
}

func TxoCoinReceiveByKeys(txVersion uint32, serializedTxOut []byte, cryptoAddress []byte, cryptoValueSecretKey []byte) (bool, uint64, error) {
	return api.TxoCoinReceiveByKeys(txVersion, serializedTxOut, cryptoAddress, cryptoValueSecretKey)
}

func GenerateCoinSerialNumberByKeys(outPoints []*OutPoint,
	serializedBlocksForRingGroup [][]byte,
	cryptoSnsks [][]byte) (serialNumbers [][]byte, err error) {
	return api.GenerateCoinSerialNumberByKeys(outPoints, serializedBlocksForRingGroup, cryptoSnsks)
}
