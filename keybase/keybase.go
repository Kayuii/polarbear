package keybase

import (
	"encoding/json"
	"github.com/coinexchain/dex/modules/alias"
	"github.com/coinexchain/dex/modules/asset"
	"github.com/coinexchain/dex/modules/bancorlite"
	"github.com/coinexchain/dex/modules/bankx"
	"github.com/coinexchain/dex/modules/comment"
	"github.com/coinexchain/dex/modules/distributionx"
	"github.com/coinexchain/dex/modules/market"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	sigType "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/go-bip39"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

const (
	mnemonicEntropySize = 256
	defaultCoinType     = 688
)

var gCdc = codec.New()

type KeyBase interface {
	CreateKey(name, password, bip39Passphrase string, account, index uint32) string
	DeleteKey(name, password string) string
	RecoverKey(name, mnemonic, password, bip39Passphrase string, account, index uint32) string
	AddKey(name, armor, passphrase string) string
	ExportKey(name, decryptPassphrase, encryptPassphrase string) string
	ListKeys() string
	GetAddress(name string) string
	GetPubKey(name string) string
	ResetPassword(name, password, newPassword string) string
	GetSigner(signerInfo string) string
	Sign(name, password, tx string) string
	SignStdTx(name, password, tx, chainId string, accountNum, sequence uint64) string
	SignAndBuildBroadcast(name, password, tx, chainId, mode string, accountNum, sequence uint64) string
}

var _ KeyBase = DefaultKeyBase{}

type DefaultKeyBase struct {
	kb   keys.Keybase
	name string
	dir  string
}

func NewDefaultKeyBase(root string) DefaultKeyBase {
	initDefaultKeyBaseConfig()
	return DefaultKeyBase{
		keys.New("keys", root),
		"keys",
		root,
	}
}

//todo: name repetition check
func (k DefaultKeyBase) CreateKey(name, password, bip39Passphrase string, account, index uint32) string {
	entropySeed, err := bip39.NewEntropy(mnemonicEntropySize)
	if err != nil {
		return ""
	}

	mnemonic, err := bip39.NewMnemonic(entropySeed[:])
	if err != nil {
		return ""
	}
	hdPath := hd.NewFundraiserParams(account, defaultCoinType, index)
	info, err := k.kb.Derive(name, mnemonic, bip39Passphrase, password, *hdPath)
	if err != nil {
		return ""
	}
	return info.GetAddress().String() + "+" + mnemonic
}

func (k DefaultKeyBase) DeleteKey(name, password string) string {
	if err := k.kb.Delete(name, password, false); err != nil {
		return err.Error()
	}
	return ""
}

func (k DefaultKeyBase) RecoverKey(name, mnemonic, password, bip39Passphrase string, account, index uint32) string {
	info, err := k.kb.CreateAccount(name, mnemonic, bip39Passphrase, password, account, index)
	if err != nil {
		return ""
	}
	return info.GetAddress().String()
}

func (k DefaultKeyBase) AddKey(name, armor, passphrase string) string {
	if err := k.kb.ImportPrivKey(name, armor, passphrase); err != nil {
		return err.Error()
	}
	//addr := k.GetAddress(name)
	//if addr == "" {
	//	return "no corresponding address"
	//}
	//levelDb, err := sdk.NewLevelDB(k.name, k.dir)
	//if err != nil {
	//	return err.Error()
	//}
	//defer levelDb.Close()
	//
	//addressSuffix := "address"
	//infoSuffix := "info"
	//addrKey := func(address types.AccAddress) []byte {
	//	return []byte(fmt.Sprintf("%s.%s", address.String(), addressSuffix))
	//}
	//infoKey := func(name string) []byte {
	//	return []byte(fmt.Sprintf("%s.%s", name, infoSuffix))
	//}
	//accAddr, err := sdk.AccAddressFromBech32(addr)
	//if err != nil {
	//	return err.Error()
	//}
	//levelDb.SetSync(addrKey(accAddr), infoKey(name))
	return ""
}

func (k DefaultKeyBase) ExportKey(name, decryptPassphrase, encryptPassphrase string) string {
	armor, err := k.kb.ExportPrivKey(name, decryptPassphrase, encryptPassphrase)
	if err != nil {
		return ""
	}
	return armor
}

func (k DefaultKeyBase) ListKeys() string {
	infos, err := k.kb.List()
	if err != nil {
		return ""
	}
	out, err := infosToJson(infos)
	if err != nil {
		return ""
	}
	return out
}

func (k DefaultKeyBase) GetAddress(name string) string {
	info, err := k.kb.Get(name)
	if err != nil {
		return ""
	}
	return info.GetAddress().String()
}

func (k DefaultKeyBase) GetPubKey(name string) string {
	info, err := k.kb.Get(name)
	if err != nil {
		return ""
	}
	benchPubKey, err := sdk.Bech32ifyAccPub(info.GetPubKey())
	if err != nil {
		return ""
	}
	return benchPubKey
}

func (k DefaultKeyBase) ResetPassword(name, password, newPassword string) string {
	f := func() (string, error) { return newPassword, nil }
	if err := k.kb.Update(name, password, f); err != nil {
		return err.Error()
	}
	return ""
}

func (k DefaultKeyBase) GetSigner(signerInfo string) string {
	var sign sigType.StdSignDoc
	err := gCdc.UnmarshalJSON([]byte(signerInfo), &sign)
	if err != nil {
		return ""
	}
	var msg sdk.Msg
	for _, m := range sign.Msgs {
		err := gCdc.UnmarshalJSON(m, &msg)
		if err != nil {
			return ""
		}
		signers := msg.GetSigners()
		if signers == nil || len(signers) == 0 {
			return ""
		}
		signer := msg.GetSigners()[0]
		info, err := k.kb.GetByAddress(signer)
		if err != nil {
			return ""
		}
		return info.GetName()
	}
	return ""
}

func (k DefaultKeyBase) Sign(name, password, tx string) string {
	sig, pub, err := k.kb.Sign(name, password, []byte(tx))
	if err != nil {
		return ""
	}
	stdSign := StdSignature{pub, sig}
	out, err := gCdc.MarshalJSON(stdSign)
	if err != nil {
		return ""
	}
	return string(out)
}

func (k DefaultKeyBase) signStdTx(name, password, tx, chainId string, accountNum, sequence uint64) (sigType.StdTx, string) {
	stdTx := sigType.StdTx{}
	err := gCdc.UnmarshalJSON([]byte(tx), &stdTx)
	if err != nil {
		return stdTx, err.Error()
	}
	var msgsBytes []json.RawMessage
	for _, msg := range stdTx.Msgs {
		msgsBytes = append(msgsBytes, json.RawMessage(msg.GetSignBytes()))
	}
	doc := sigType.StdSignDoc{
		AccountNumber: accountNum,
		ChainID:       chainId,
		Fee:           stdTx.Fee.Bytes(),
		Memo:          stdTx.Memo,
		Msgs:          msgsBytes,
		Sequence:      sequence,
	}
	bz, err := gCdc.MarshalJSON(doc)
	if err != nil {
		return stdTx, "marshal doc error"
	}
	ret, err := sdk.SortJSON(bz)
	if err != nil {
		return stdTx, "sort doc error"
	}
	out := k.Sign(name, password, string(ret))
	return stdTx, out
}

func (k DefaultKeyBase) SignStdTx(name, password, tx, chainId string, accountNum, sequence uint64) string {
	_, out := k.signStdTx(name, password, tx, chainId, accountNum, sequence)
	return out
}

func (k DefaultKeyBase) SignAndBuildBroadcast(name, password, tx, chainId, mode string, accountNum, sequence uint64) string {
	stdTx, out := k.signStdTx(name, password, tx, chainId, accountNum, sequence)
	sig := sigType.StdSignature{}
	err := gCdc.UnmarshalJSON([]byte(out), &sig)
	if err != nil {
		return err.Error()
	}
	stdTx.Signatures = []sigType.StdSignature{sig}
	req := rest.BroadcastReq{
		Tx:   stdTx,
		Mode: mode,
	}
	ret, err := gCdc.MarshalJSON(req)
	if err != nil {
		return err.Error()
	}
	return string(ret)
}

func initDefaultKeyBaseConfig() {
	initCodec()
	bench32MainPrefix := "coinex"
	bench32PrefixAccAddr := bench32MainPrefix
	// bench32PrefixAccPub defines the bench32 prefix of an account's public key
	bench32PrefixAccPub := bench32MainPrefix + types.PrefixPublic
	// bench32PrefixValAddr defines the bench32 prefix of a validator's operator address
	bench32PrefixValAddr := bench32MainPrefix + types.PrefixValidator + types.PrefixOperator
	// bench32PrefixValPub defines the bench32 prefix of a validator's operator public key
	bench32PrefixValPub := bench32MainPrefix + types.PrefixValidator + types.PrefixOperator + types.PrefixPublic
	// bench32PrefixConsAddr defines the bench32 prefix of a consensus node address
	bench32PrefixConsAddr := bench32MainPrefix + types.PrefixValidator + types.PrefixConsensus
	// bench32PrefixConsPub defines the bench32 prefix of a consensus node public key
	bench32PrefixConsPub := bench32MainPrefix + types.PrefixValidator + types.PrefixConsensus + types.PrefixPublic

	config := types.GetConfig()
	config.SetCoinType(defaultCoinType)
	config.SetBech32PrefixForAccount(bench32PrefixAccAddr, bench32PrefixAccPub)
	config.SetBech32PrefixForValidator(bench32PrefixValAddr, bench32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(bench32PrefixConsAddr, bench32PrefixConsPub)
	config.Seal()
}

func initCodec() {
	gCdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	gCdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	gCdc.RegisterInterface((*sdk.Msg)(nil), nil)
	gCdc.RegisterConcrete(secp256k1.PubKeySecp256k1{}, "tendermint/PubKeySecp256k1", nil)
	gCdc.RegisterConcrete(secp256k1.PrivKeySecp256k1{}, "tendermint/PrivKeySecp256k1", nil)
	gCdc.RegisterConcrete(sigType.StdTx{}, "auth/StdTx", nil)
	//alias
	gCdc.RegisterConcrete(alias.MsgAliasUpdate{}, "alias/MsgAliasUpdate", nil)
	//asset
	gCdc.RegisterConcrete(asset.MsgIssueToken{}, "asset/MsgIssueToken", nil)
	gCdc.RegisterConcrete(asset.MsgTransferOwnership{}, "asset/MsgTransferOwnership", nil)
	gCdc.RegisterConcrete(asset.MsgMintToken{}, "asset/MsgMintToken", nil)
	gCdc.RegisterConcrete(asset.MsgBurnToken{}, "asset/MsgBurnToken", nil)
	gCdc.RegisterConcrete(asset.MsgForbidToken{}, "asset/MsgForbidToken", nil)
	gCdc.RegisterConcrete(asset.MsgUnForbidToken{}, "asset/MsgUnForbidToken", nil)
	gCdc.RegisterConcrete(asset.MsgAddTokenWhitelist{}, "asset/MsgAddTokenWhitelist", nil)
	gCdc.RegisterConcrete(asset.MsgRemoveTokenWhitelist{}, "asset/MsgRemoveTokenWhitelist", nil)
	gCdc.RegisterConcrete(asset.MsgForbidAddr{}, "asset/MsgForbidAddr", nil)
	gCdc.RegisterConcrete(asset.MsgUnForbidAddr{}, "asset/MsgUnForbidAddr", nil)
	gCdc.RegisterConcrete(asset.MsgModifyTokenInfo{}, "asset/MsgModifyTokenInfo", nil)
	//bankx
	gCdc.RegisterConcrete(bankx.MsgSetMemoRequired{}, "bankx/MsgSetMemoRequired", nil)
	gCdc.RegisterConcrete(bankx.MsgSend{}, "bankx/MsgSend", nil)
	gCdc.RegisterConcrete(bankx.MsgMultiSend{}, "bankx/MsgMultiSend", nil)
	//bancor
	gCdc.RegisterConcrete(bancorlite.MsgBancorInit{}, "bancorlite/MsgBancorInit", nil)
	gCdc.RegisterConcrete(bancorlite.MsgBancorTrade{}, "bancorlite/MsgBancorTrade", nil)
	gCdc.RegisterConcrete(bancorlite.MsgBancorCancel{}, "bancorlite/MsgBancorCancel", nil)
	//comment
	gCdc.RegisterConcrete(comment.MsgCommentToken{}, "comment/MsgCommentToken", nil)
	//distribution
	gCdc.RegisterConcrete(distributionx.MsgDonateToCommunityPool{}, "distrx/MsgDonateToCommunityPool", nil)
	//gov
	gCdc.RegisterConcrete(gov.MsgSubmitProposal{}, "cosmos-sdk/MsgSubmitProposal", nil)
	gCdc.RegisterConcrete(gov.MsgDeposit{}, "cosmos-sdk/MsgDeposit", nil)
	gCdc.RegisterConcrete(gov.MsgVote{}, "cosmos-sdk/MsgVote", nil)
	//market
	gCdc.RegisterConcrete(market.MsgCreateTradingPair{}, "market/MsgCreateTradingPair", nil)
	gCdc.RegisterConcrete(market.MsgCreateOrder{}, "market/MsgCreateOrder", nil)
	gCdc.RegisterConcrete(market.MsgCancelOrder{}, "market/MsgCancelOrder", nil)
	gCdc.RegisterConcrete(market.MsgCancelTradingPair{}, "market/MsgCancelTradingPair", nil)
	gCdc.RegisterConcrete(market.MsgModifyPricePrecision{}, "market/MsgModifyPricePrecision", nil)
	//stake
	gCdc.RegisterConcrete(staking.MsgCreateValidator{}, "cosmos-sdk/MsgCreateValidator", nil)
	gCdc.RegisterConcrete(staking.MsgEditValidator{}, "cosmos-sdk/MsgEditValidator", nil)
	gCdc.RegisterConcrete(staking.MsgDelegate{}, "cosmos-sdk/MsgDelegate", nil)
	gCdc.RegisterConcrete(staking.MsgUndelegate{}, "cosmos-sdk/MsgUndelegate", nil)
	gCdc.RegisterConcrete(staking.MsgBeginRedelegate{}, "cosmos-sdk/MsgBeginRedelegate", nil)
}

// NewFundraiserParams creates a BIP 44 parameter object from the params:
// m / 44' / coinType' / account' / 0 / address_index
// The fixed parameters (purpose', coin_type', and change) are determined by what was used in the fundraiser.
//func NewFundraiserParams(account, coinType, addressIdx uint32) *BIP44Params {
//	return NewParams(44, coinType, account, false, addressIdx)
//}

func GetAddressFromEntropy(entropy []byte) (string, string, error) {
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", mnemonic, err
	}

	DefaultBIP39Passphrase := ""
	seed := bip39.NewSeed(mnemonic, DefaultBIP39Passphrase)
	fullHdPath := hd.NewFundraiserParams(0, defaultCoinType, 0) //account=0 addressIdx=0
	masterPriv, ch := hd.ComputeMastersFromSeed(seed)
	derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, ch, fullHdPath.String())
	pubk := secp256k1.PrivKeySecp256k1(derivedPriv).PubKey()
	addr := pubk.Address()
	acc := sdk.AccAddress(addr)
	return acc.String(), mnemonic, nil
}

func infosToJson(infos []keys.Info) (string, error) {
	kos, err := keys.Bech32KeysOutput(infos)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(kos)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type StdSignature struct {
	crypto.PubKey `json:"pub_key"`
	Signature     []byte `json:"signature"`
}
