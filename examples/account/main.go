package main

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
)

func main() {
	account, _ /*seedBytes*/, err := abelian.NewAccount(abelian.AccountPrivacyLevelPseudonym)
	if err != nil {
		panic(fmt.Errorf("fail to generate account"))
	}
	viewAccount := account.ViewAccount()

}
