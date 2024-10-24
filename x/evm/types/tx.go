package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sei-protocol/sei-chain/x/evm/types/ethtx"
)

func NewTxWithData(tx *ethtypes.Transaction) *MsgEVMTransaction {
	typedTxData, err := ethtx.NewTxDataFromTx(tx)
	if err != nil {
		panic(err)
	}

	msg, err := NewMsgEVMTransaction(typedTxData)
	if err != nil {
		panic(err)
	}

	return msg
}

// NewTx returns a reference to a new Ethereum transaction message.
func NewTx(
	chainID *big.Int, nonce uint64, to *common.Address, amount *big.Int,
	gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, input []byte, accesses *ethtypes.AccessList,
) *MsgEVMTransaction {
	return newMsgEthereumTx(chainID, nonce, to, amount, gasLimit, gasPrice, gasFeeCap, gasTipCap, input, accesses)
}

// NewTxContract returns a reference to a new Ethereum transaction
// message designated for contract creation.
func NewTxContract(
	chainID *big.Int,
	nonce uint64,
	amount *big.Int,
	gasLimit uint64,
	gasPrice, gasFeeCap, gasTipCap *big.Int,
	input []byte,
	accesses *ethtypes.AccessList,
) *MsgEVMTransaction {
	return newMsgEthereumTx(chainID, nonce, nil, amount, gasLimit, gasPrice, gasFeeCap, gasTipCap, input, accesses)
}

func newMsgEthereumTx(
	chainID *big.Int, nonce uint64, to *common.Address, amount *big.Int,
	gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, input []byte, accesses *ethtypes.AccessList,
) *MsgEVMTransaction {
	var txData ethtypes.TxData

	switch {
	case gasFeeCap != nil:
		var accessList ethtypes.AccessList
		if accesses != nil {
			accessList = *accesses
		}
		txData = &ethtypes.DynamicFeeTx{
			ChainID:    chainID,
			Nonce:      nonce,
			To:         to,
			Value:      amount,
			Gas:        gasLimit,
			GasTipCap:  gasTipCap,
			GasFeeCap:  gasFeeCap,
			Data:       input,
			AccessList: accessList,
		}
	case accesses != nil:
		txData = &ethtypes.AccessListTx{
			ChainID:    chainID,
			Nonce:      nonce,
			To:         to,
			Value:      amount,
			Gas:        gasLimit,
			GasPrice:   gasPrice,
			Data:       input,
			AccessList: *accesses,
		}
	default:
		txData = &ethtypes.LegacyTx{
			Nonce:    nonce,
			To:       to,
			Value:    amount,
			Gas:      gasLimit,
			GasPrice: gasPrice,
			Data:     input,
		}
	}

	return NewTxWithData(ethtypes.NewTx(txData))
}
