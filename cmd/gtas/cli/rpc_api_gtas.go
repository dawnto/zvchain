//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cli

import (
	"encoding/json"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"strings"
)

// RpcGtasImpl provides rpc service for users to interact with remote nodes
type RpcGtasImpl struct {
}

func (api *RpcGtasImpl) Namespace() string {
	return "Gtas"
}

func (api *RpcGtasImpl) Version() string {
	return "1"
}

func successResult(data interface{}) (*Result, error) {
	return &Result{
		Message: "success",
		Data:    data,
		Status:  0,
	}, nil
}
func failResult(err string) (*Result, error) {
	return &Result{
		Message: err,
		Data:    nil,
		Status:  -1,
	}, nil
}

// Tx is user transaction interface, used for sending transaction to the node
func (api *RpcGtasImpl) Tx(txRawjson string) (*Result, error) {
	var txRaw = new(txRawData)
	if err := json.Unmarshal([]byte(txRawjson), txRaw); err != nil {
		return failResult(err.Error())
	}
	if !validateTxType(txRaw.TxType) {
		return failResult("Not supported txType")
	}
	if txRaw.TxType == types.TransactionTypeTransfer || txRaw.TxType == types.TransactionTypeContractCall {
		if !validateAddress(strings.TrimSpace(txRaw.Target)) {
			return failResult("Wrong target address format")
		}
	}
	trans := txRawToTransaction(txRaw)

	trans.Hash = trans.GenHash()

	if err := sendTransaction(trans); err != nil {
		return failResult(err.Error())
	}

	return successResult(trans.Hash.Hex())
}

// Balance is query balance interface
func (api *RpcGtasImpl) Balance(account string) (*Result, error) {
	if !validateAddress(strings.TrimSpace(account)) {
		return failResult("Wrong account address format")
	}
	b := core.BlockChainImpl.GetBalance(common.HexToAddress(account))

	balance := common.RA2TAS(b.Uint64())
	return &Result{
		Message: fmt.Sprintf("The balance of account: %s is %v TAS", account, balance),
		Data:    balance,
	}, nil
}

// BlockHeight query block height
func (api *RpcGtasImpl) BlockHeight() (*Result, error) {
	height := core.BlockChainImpl.QueryTopBlock().Height
	return successResult(height)
}

// GroupHeight query group height
func (api *RpcGtasImpl) GroupHeight() (*Result, error) {
	height := core.GroupChainImpl.Height()
	return successResult(height)
}

func (api *RpcGtasImpl) GetBlockByHeight(height uint64) (*Result, error) {
	b := core.BlockChainImpl.QueryBlockByHeight(height)
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *RpcGtasImpl) GetBlockByHash(hash string) (*Result, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return failResult("Wrong hash format")
	}
	b := core.BlockChainImpl.QueryBlockByHash(common.HexToHash(hash))
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBH := core.BlockChainImpl.QueryBlockHeaderByHash(bh.PreHash)
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *RpcGtasImpl) MinerInfo(addr string) (*Result, error) {
	if !validateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong account address format")
	}
	morts := make([]MortGage, 0)
	id := common.HexToAddress(addr).Bytes()
	heavyInfo := core.MinerManagerImpl.GetMinerByID(id, types.MinerTypeHeavy, nil)
	if heavyInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(heavyInfo))
	}
	lightInfo := core.MinerManagerImpl.GetMinerByID(id, types.MinerTypeLight, nil)
	if lightInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(lightInfo))
	}
	return successResult(morts)
}

func (api *RpcGtasImpl) TransDetail(h string) (*Result, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return failResult("Wrong hash format")
	}
	tx := core.BlockChainImpl.GetTransactionByHash(false, true, common.HexToHash(h))

	if tx != nil {
		trans := convertTransaction(tx)
		return successResult(trans)
	}
	return successResult(nil)
}

func (api *RpcGtasImpl) Nonce(addr string) (*Result, error) {
	if !validateAddress(strings.TrimSpace(addr)) {
		return failResult("Wrong account address format")
	}
	address := common.HexToAddress(addr)
	// user will see the nonce as db nonce +1, so that user can use it directly when send a transaction
	nonce := core.BlockChainImpl.GetNonce(address) + 1
	return successResult(nonce)
}

func (api *RpcGtasImpl) TxReceipt(h string) (*Result, error) {
	if !validateHash(strings.TrimSpace(h)) {
		return failResult("Wrong hash format")
	}
	hash := common.HexToHash(h)
	rc := core.BlockChainImpl.GetTransactionPool().GetReceipt(hash)
	if rc != nil {
		tx := core.BlockChainImpl.GetTransactionByHash(false, true, hash)
		return successResult(&core.ExecutedTransaction{
			Receipt:     rc,
			Transaction: tx,
		})
	}
	return failResult("tx not exist")
}

// ViewAccount is used for querying account information
func (api *RpcGtasImpl) ViewAccount(hash string) (*Result, error) {
	if !validateHash(strings.TrimSpace(hash)) {
		return failResult("Wrong hash format")
	}
	accoundDb := core.BlockChainImpl.LatestStateDB()
	if accoundDb == nil {
		return nil, nil
	}
	address := common.HexToAddress(hash)
	if !accoundDb.Exist(address) {
		return failResult("Account not Exist!")
	}
	account := ExplorerAccount{}
	account.Balance = accoundDb.GetBalance(address)
	account.Nonce = accoundDb.GetNonce(address)
	account.CodeHash = accoundDb.GetCodeHash(address).Hex()
	account.Code = string(accoundDb.GetCode(address)[:])
	account.Type = 0
	if len(account.Code) > 0 {
		account.Type = 1
		account.StateData = make(map[string]interface{})

		iter := accoundDb.DataIterator(common.HexToAddress(hash), "")
		for iter.Next() {
			k := string(iter.Key[:])
			v := string(iter.Value[:])
			account.StateData[k] = v

		}
	}
	return successResult(account)
}

// PledgeDetail query the pledge details of the given account.
// from: the miner address who launches the pledge, optional.
// to: the miner address who was pledged, required.
// All pledge detail will be returned if the from param is empty
func (api *RpcGtasImpl) PledgeDetail(from, to string) (*Result, error) {
	return &Result{}, nil
}
