package wallet

import (
	"github.com/coinexchain/polarbear/keybase"
)

var Api Wallet

type Wallet struct {
	keybase.KeyBase
}

func BearInit(root string) {
	Api.KeyBase = keybase.NewDefaultKeyBase(root)
}

func CreateKey(name, password, bip39Passphrase string, account, index uint32) string {
	return Api.CreateKey(name, password, bip39Passphrase, account, index)
}

func DeleteKey(name, password string) string {
	return Api.DeleteKey(name, password)
}

func RecoverKey(name, mnemonic, password, bip39Passphrase string, account, index uint32) string {
	return Api.RecoverKey(name, mnemonic, password, bip39Passphrase, account, index)
}

func AddKey(name, armor, passphrase string) string {
	return Api.AddKey(name, armor, passphrase)
}

func ExportKey(name, decryptPassphrase, encryptPassphrase string) string {
	return Api.ExportKey(name, decryptPassphrase, encryptPassphrase)
}

func ListKeys() string {
	return Api.ListKeys()
}

func ResetPassword(name, password, newPassword string) string {
	return Api.ResetPassword(name, password, newPassword)
}

func GetAddress(name string) string {
	return Api.GetAddress(name)
}

func GetPubKey(name string) string {
	return Api.GetPubKey(name)
}

func GetSigner(signerInfo string) string {
	return Api.GetSigner(signerInfo)
}

func Sign(name, password, tx string) string {
	return Api.Sign(name, password, tx)
}

func SignStdTx(name, password, tx, chainId string, accountNum, sequence uint64) string {
	return Api.SignStdTx(name, password, tx, chainId, accountNum, sequence)
}

func SignAndBuildBroadcast(name, password, tx, chainId, mode string, accountNum, sequence uint64) string {
	return Api.SignAndBuildBroadcast(name, password, tx, chainId, mode, accountNum, sequence)
}
