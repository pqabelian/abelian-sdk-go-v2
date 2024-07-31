package main

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

func GenerateAddress(cryptoScheme crypto.CryptoScheme, privacyLevel crypto.PrivacyLevel) {
	seeds, err := crypto.GenerateSeed(cryptoScheme, privacyLevel)
	if err != nil {
		panic(fmt.Errorf("fail to generate crypto seed: %v", err))
	}

	serializedSeeds, err := seeds.Serialize()
	if err != nil {
		panic(fmt.Errorf("fail to serialized generated seed: %v", err))
	}

	// Keep it in a safe place and never leak it to others.
	// Only thing u need to remember is the seed, which will produce all the subsequent information.
	cryptoKeysAndAddress, err := crypto.GenerateCryptoKeysAndAddressBySeedBytes(serializedSeeds)
	if err != nil {
		panic(fmt.Errorf("fail to generate address: %v", err))
	}
	fmt.Printf("%s: %d bytes | %x\n", seeds.Type(), len(serializedSeeds), serializedSeeds)
	fmt.Printf("CryptoAddress: %d bytes | %x\n", len(cryptoKeysAndAddress.CryptoAddress.Data()), cryptoKeysAndAddress.CryptoAddress.Data())
	fmt.Printf("SpendSecretKey: %d bytes | %x\n", len(cryptoKeysAndAddress.SpendSecretKey), cryptoKeysAndAddress.SpendSecretKey)
	fmt.Printf("SerialNoSecretKey: %d bytes | %x\n", len(cryptoKeysAndAddress.SerialNoSecretKey), cryptoKeysAndAddress.SerialNoSecretKey)
	fmt.Printf("ViewSecretKey: %d bytes | %x\n", len(cryptoKeysAndAddress.ViewSecretKey), cryptoKeysAndAddress.ViewSecretKey)
	fmt.Printf("DetectorKey: %d bytes | %x\n", len(cryptoKeysAndAddress.DetectorKey), cryptoKeysAndAddress.DetectorKey)
}

func main() {
	for _, x := range []struct {
		crypto.CryptoScheme
		crypto.PrivacyLevel
	}{
		{
			crypto.CryptoSchemePQRingCTX,
			crypto.PrivacyLevelFullPrivacyRand,
		},
		{
			crypto.CryptoSchemePQRingCTX,
			crypto.PrivacyLevelPseudonym,
		},
	} {
		fmt.Println("==============")
		fmt.Printf("crypto scheme %d, privacy level %d \n", x.CryptoScheme, x.PrivacyLevel)
		GenerateAddress(x.CryptoScheme, x.PrivacyLevel)
		fmt.Println("==============")
	}
}
