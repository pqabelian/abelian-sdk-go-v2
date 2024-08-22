package main

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func main() {
	networkID := common.GetNetworkID()

	privacyLevels := []abelian.AccountPrivacyLevel{
		abelian.AccountPrivacyLevelFullPrivacy,
		abelian.AccountPrivacyLevelPseudonym,
	}

	for _, privacyLevel := range privacyLevels {
		account, err := abelian.NewAccount(networkID, privacyLevel)
		if err != nil {
			panic(fmt.Errorf("fail to generate account:%v", err))
		}

		spendKey := account.SpendKeyMaterial()
		snKeySeed, valueKeySeed, detectKey := account.ViewKeyMaterial()
		accountID, err := database.InsertAccount(networkID, privacyLevel, spendKey, snKeySeed, valueKeySeed, detectKey)
		if err != nil {
			panic(err)
		}

		loadedAccount, err := database.LoadAccountByID(accountID)
		if err != nil {
			panic(err)
		}
		address, err := loadedAccount.GenerateAbelAddress()
		fmt.Printf("%x\n", address)
	}
}
