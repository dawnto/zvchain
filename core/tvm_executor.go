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

package core

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/storage/account"
	"github.com/zvchain/zvchain/storage/vm"
	"github.com/zvchain/zvchain/tvm"
)

const TransactionGasCost uint64 = 1000
const CodeBytePrice = 0.3814697265625
const MaxCastBlockTime = time.Second * 3

type TVMExecutor struct {
	bc BlockChain
}

func NewTVMExecutor(bc BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc: bc,
	}
}

// Execute executes all types transactions and returns the receipts
func (executor *TVMExecutor) Execute(accountdb *account.AccountDB, bh *types.BlockHeader, txs []*types.Transaction, pack bool, ts *common.TimeStatCtx) (state common.Hash, evits []common.Hash, executed []*types.Transaction, recps []*types.Receipt, err error) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	castor := common.BytesToAddress(bh.Castor)

	for _, transaction := range txs {
		if pack && time.Since(beginTime).Seconds() > float64(MaxCastBlockTime) {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.Hex())
			break
		}
		var (
			success           = false
			contractAddress   common.Address
			logs              []*types.Log
			cumulativeGasUsed uint64
			gasUsed           *types.BigInt
		)

		if !transaction.IsBonus() {
			if err = transaction.BoundCheck(); err != nil {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			intriGas, txErr := intrinsicGas(transaction)
			if txErr != nil {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}
			gasUsed = intriGas
			if !executor.validateNonce(accountdb, transaction) {
				evictedTxs = append(evictedTxs, transaction.Hash)
				continue
			}

			switch transaction.Type {
			case types.TransactionTypeTransfer:
				success, _, cumulativeGasUsed = executor.executeTransferTx(accountdb, transaction, castor, gasUsed)
			case types.TransactionTypeContractCreate:
				success, _, cumulativeGasUsed, contractAddress = executor.executeContractCreateTx(accountdb, transaction, castor, bh, gasUsed)
			case types.TransactionTypeContractCall:
				success, _, cumulativeGasUsed, logs = executor.executeContractCallTx(accountdb, transaction, castor, bh, gasUsed)
			case types.TransactionTypeMinerApply:
				success = executor.executeMinerApplyTx(accountdb, transaction, bh.Height, castor, gasUsed)
			case types.TransactionTypeMinerAbort:
				success = executor.executeMinerAbortTx(accountdb, transaction, bh.Height, castor, gasUsed)
			case types.TransactionTypeMinerRefund:
				success = executor.executeMinerRefundTx(accountdb, transaction, bh.Height, castor, gasUsed)
			case types.TransactionTypeMinerCancelStake:
				success = executor.executeMinerCancelStakeTx(accountdb, transaction, bh.Height, castor, gasUsed)
			case types.TransactionTypeMinerStake:
				success = executor.executeMinerStakeTx(accountdb, transaction, bh.Height, castor, gasUsed)
			}
		} else {
			success = executor.executeBonusTx(accountdb, transaction, castor)
			if !success {
				evictedTxs = append(evictedTxs, transaction.Hash)
				// Failed bonus tx should not be included in block
				continue
			}
		}

		idx := len(transactions)
		transactions = append(transactions, transaction)
		receipt := types.NewReceipt(nil, !success, cumulativeGasUsed)
		receipt.Logs = logs
		receipt.TxHash = transaction.Hash
		receipt.ContractAddress = contractAddress
		receipt.TxIndex = uint16(idx)
		receipt.Height = bh.Height
		receipts = append(receipts, receipt)
		//errs[i] = err
		if transaction.Source != nil {
			accountdb.SetNonce(*transaction.Source, transaction.Nonce)
		}

	}
	//ts.AddStat("executeLoop", time.Since(b))
	accountdb.AddBalance(castor, executor.bc.GetConsensusHelper().ProposalBonus())

	state = accountdb.IntermediateRoot(true)
	return state, evictedTxs, transactions, receipts, nil
}

func (executor *TVMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	if transaction.Type == types.TransactionTypeBonus || IsTestTransaction(transaction) {
		return true
	}
	nonce := accountdb.GetNonce(*transaction.Source)
	if transaction.Nonce != nonce+1 {
		Logger.Infof("Tx nonce error! Hash:%s,Source:%s,expect nonce:%d,real nonce:%d ", transaction.Hash.Hex(), transaction.Source.Hex(), nonce+1, transaction.Nonce)
		return false
	}
	return true
}

func (executor *TVMExecutor) executeTransferTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, gasUsed *types.BigInt) (success bool, err *types.TransactionError, cumulativeGasUsed uint64) {
	success = false

	amount := transaction.Value.Value()

	gasFee := new(types.BigInt).Mul(gasUsed.Value(), transaction.GasPrice.Value())
	if canTransfer(accountdb, *transaction.Source, amount, gasFee) {
		transfer(accountdb, *transaction.Source, *transaction.Target, amount)
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)
		cumulativeGasUsed = gasUsed.Uint64()
		success = true
	} else {
		err = types.TxErrorBalanceNotEnough
	}
	return success, err, cumulativeGasUsed
}

func (executor *TVMExecutor) executeContractCreateTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, bh *types.BlockHeader, gasUsed *types.BigInt) (success bool, err *types.TransactionError, cumulativeGasUsed uint64, contractAddress common.Address) {
	success = false

	var txErr *types.TransactionError

	gasLimit := transaction.GasLimit
	gasLimitFee := new(types.BigInt).Mul(transaction.GasLimit.Value(), transaction.GasPrice.Value())

	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), gasLimitFee) {
		accountdb.SubBalance(*transaction.Source, gasLimitFee)
		controller := tvm.NewController(accountdb, BlockChainImpl, bh, transaction, gasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl, GroupChainImpl)
		snapshot := controller.AccountDB.Snapshot()
		contractAddress, txErr = createContract(accountdb, transaction)
		if txErr != nil {
			Logger.Debugf("ContractCreate tx %s execute error:%s ", transaction.Hash.Hex(), txErr.Message)
			controller.AccountDB.RevertToSnapshot(snapshot)
		} else {
			contract := tvm.LoadContract(contractAddress)
			err := controller.Deploy(contract)
			if err != nil {
				txErr = types.NewTransactionError(types.TVMExecutedError, err.Error())
				controller.AccountDB.RevertToSnapshot(snapshot)
				Logger.Debugf("Contract deploy failed! Tx hash:%s, contract addr:%s errorCode:%d errorMsg%s",
					transaction.Hash.Hex(), contractAddress.Hex(), types.TVMExecutedError, err.Error())
			} else {
				success = true
				Logger.Debugf("Contract create success! Tx hash:%s, contract addr:%s", transaction.Hash.Hex(), contractAddress.Hex())
			}
		}
		gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
		allUsed := new(big.Int).Sub(gasLimit.Value(), gasLeft)

		returnFee := new(big.Int).Mul(gasLeft, transaction.GasPrice.Value())
		allFee := new(big.Int).Mul(allUsed, transaction.GasPrice.Value())
		accountdb.AddBalance(*transaction.Source, returnFee)
		accountdb.AddBalance(castor, allFee)

		cumulativeGasUsed = allUsed.Uint64()

	} else {
		success = false
		txErr = types.TxErrorBalanceNotEnough
		Logger.Infof("ContractCreate balance not enough! transaction %s source %s  ", transaction.Hash.Hex(), transaction.Source.Hex())
	}
	Logger.Debugf("TVMExecutor Execute ContractCreate Transaction %s,success:%t", transaction.Hash.Hex(), success)
	return success, txErr, cumulativeGasUsed, contractAddress
}

func (executor *TVMExecutor) executeContractCallTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, bh *types.BlockHeader, gasUsed *types.BigInt) (success bool, err *types.TransactionError, cumulativeGasUsed uint64, logs []*types.Log) {
	success = false
	transferAmount := transaction.Value.Value()

	gasLimit := transaction.GasLimit
	gasLimitFee := new(types.BigInt).Mul(transaction.GasLimit.Value(), transaction.GasPrice.Value())

	if canTransfer(accountdb, *transaction.Source, transferAmount, gasLimitFee) {
		accountdb.SubBalance(*transaction.Source, gasLimitFee)
		controller := tvm.NewController(accountdb, BlockChainImpl, bh, transaction, gasUsed.Uint64(), common.GlobalConf.GetString("tvm", "pylib", "lib"), MinerManagerImpl, GroupChainImpl)
		contract := tvm.LoadContract(*transaction.Target)
		if contract.Code == "" {
			err = types.NewTransactionError(types.TxErrorCodeNoCode, fmt.Sprintf(types.NoCodeErrorMsg, *transaction.Target))

		} else {
			snapshot := controller.AccountDB.Snapshot()
			success, logs, err = controller.ExecuteABI(transaction.Source, contract, string(transaction.Data))
			if !success {
				controller.AccountDB.RevertToSnapshot(snapshot)
			} else {
				success = true
				transfer(accountdb, *transaction.Source, *contract.ContractAddress, transferAmount)
			}
		}
		gasLeft := new(big.Int).SetUint64(controller.GetGasLeft())
		allUsed := new(big.Int).Sub(gasLimit.Value(), gasLeft)

		returnFee := new(big.Int).Mul(gasLeft, transaction.GasPrice.Value())
		allFee := new(big.Int).Mul(allUsed, transaction.GasPrice.Value())
		accountdb.AddBalance(*transaction.Source, returnFee)
		accountdb.AddBalance(castor, allFee)

		cumulativeGasUsed = allUsed.Uint64()
	} else {
		err = types.TxErrorBalanceNotEnough
	}
	Logger.Debugf("TVMExecutor Execute ContractCall Transaction %s,success:%t", transaction.Hash.Hex(), success)
	return success, err, cumulativeGasUsed, logs
}

func (executor *TVMExecutor) executeBonusTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address) (success bool) {
	success = false
	if executor.bc.GetBonusManager().contain(transaction.Data, accountdb) == false {
		reader := bytes.NewReader(transaction.ExtraData)
		groupID := make([]byte, common.GroupIDLength)
		addr := make([]byte, common.AddressLength)
		if n, _ := reader.Read(groupID); n != common.GroupIDLength {
			Logger.Errorf("TVMExecutor Read GroupID Fail")
			return success
		}
		for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
			if n != common.AddressLength {
				Logger.Errorf("TVMExecutor Bonus Addr Size:%d Invalid", n)
				break
			}
			address := common.BytesToAddress(addr)
			accountdb.AddBalance(address, transaction.Value.Value())
		}
		executor.bc.GetBonusManager().put(transaction.Data, transaction.Hash[:], accountdb)
		accountdb.AddBalance(castor, executor.bc.GetConsensusHelper().PackBonus())
		success = true
	}
	return success
}

func (executor *TVMExecutor) executeMinerApplyTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool) {
	Logger.Debugf("Execute miner apply tx:%s,source: %v\n", transaction.Hash.Hex(), transaction.Source.Hex())
	success = false
	if transaction.Data == nil {
		Logger.Debugf("TVMExecutor Execute MinerApply Fail(Tx data is nil) Source:%s Height:%d", transaction.Source.Hex(), height)
		return success
	}

	gasFee := new(types.BigInt).Mul(transaction.GasPrice.Value(), gasUsed.Value())

	var miner = MinerManagerImpl.Transaction2Miner(transaction)
	miner.ID = transaction.Source[:]
	amount := new(big.Int).SetUint64(miner.Stake)

	if canTransfer(accountdb, *transaction.Source, amount, gasFee) {
		mExist := MinerManagerImpl.GetMinerByID(transaction.Source[:], miner.Type, accountdb)
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)

		if mExist != nil {
			if mExist.Status != types.MinerStatusNormal {
				if mExist.Type == types.MinerTypeLight && (mExist.Stake+miner.Stake) < common.VerifyStake {
					Logger.Debugf("TVMExecutor Execute MinerApply Fail((mExist.Stake + miner.Stake) < common.VerifyStake) Source:%s Height:%d", transaction.Source.Hex(), height)
					return
				}
				snapshot := accountdb.Snapshot()
				if MinerManagerImpl.activateAndAddStakeMiner(miner, accountdb, height) &&
					MinerManagerImpl.AddStakeDetail(miner.ID, miner, miner.Stake, accountdb) {
					accountdb.SubBalance(*transaction.Source, amount)
					Logger.Debugf("TVMExecutor Execute MinerApply success(activate) Source %s", transaction.Source.Hex())
					success = true
				} else {
					accountdb.RevertToSnapshot(snapshot)
				}
			} else {
				Logger.Debugf("TVMExecutor Execute MinerApply Fail(Already Exist) Source %s", transaction.Source.Hex())
			}
			return
		}
		miner.ApplyHeight = height
		miner.Status = types.MinerStatusNormal
		if MinerManagerImpl.addMiner(transaction.Source[:], miner, accountdb) > 0 &&
			MinerManagerImpl.AddStakeDetail(miner.ID, miner, miner.Stake, accountdb) {
			accountdb.SubBalance(*transaction.Source, amount)
			Logger.Debugf("TVMExecutor Execute MinerApply Success Source:%s Height:%d", transaction.Source.Hex(), height)
			success = true
		}
	} else {
		Logger.Debugf("TVMExecutor Execute MinerApply Fail(Balance Not Enough) Source:%s Height:%d", transaction.Source.Hex(), height)
	}
	return success
}

func (executor *TVMExecutor) executeMinerStakeTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool) {
	Logger.Debugf("Execute miner Stake tx:%s,source: %v\n", transaction.Hash.Hex(), transaction.Source.Hex())
	success = false
	if transaction.Data == nil {
		Logger.Debugf("TVMExecutor Execute Miner Stake Fail(Tx data is nil) Source:%s Height:%d", transaction.Source.Hex(), height)
		return
	}

	gasFee := new(types.BigInt).Mul(transaction.GasPrice.Value(), gasUsed.Value())

	var _type, id, value = MinerManagerImpl.Transaction2MinerParams(transaction)
	amount := new(big.Int).SetUint64(value)
	if canTransfer(accountdb, *transaction.Source, amount, gasFee) {
		mExist := MinerManagerImpl.GetMinerByID(id, _type, accountdb)
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)
		if mExist == nil {
			success = false
			Logger.Debugf("TVMExecutor Execute Miner Stake Fail(Do not exist this Miner) Source:%s Height:%d", transaction.Source.Hex(), height)
		} else {
			snapshot := accountdb.Snapshot()
			if MinerManagerImpl.AddStake(mExist.ID, mExist, value, accountdb) && MinerManagerImpl.AddStakeDetail(transaction.Source[:], mExist, value, accountdb) {
				Logger.Debugf("TVMExecutor Execute MinerUpdate Success Source:%s Height:%d", transaction.Source.Hex(), height)
				accountdb.SubBalance(*transaction.Source, amount)
				success = true
			} else {
				accountdb.RevertToSnapshot(snapshot)
			}
		}
	} else {
		Logger.Debugf("TVMExecutor Execute Miner Stake Fail(Balance Not Enough) Source:%s Height:%d", transaction.Source.Hex(), height)
	}
	return success
}

func (executor *TVMExecutor) executeMinerCancelStakeTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool) {
	Logger.Debugf("Execute miner cancel pledge tx:%s,source: %v\n", transaction.Hash.Hex(), transaction.Source.Hex())
	success = false
	if transaction.Data == nil {
		Logger.Debugf("TVMExecutor Execute MinerCancelStake Fail(Tx data is nil) Source:%s Height:%d", transaction.Source.Hex(), height)
		return
	}

	gasFee := new(types.BigInt).Mul(transaction.GasPrice.Value(), gasUsed.Value())

	var _type, id, value = MinerManagerImpl.Transaction2MinerParams(transaction)
	if canTransfer(accountdb, *transaction.Source, big.NewInt(0), gasFee) {
		mExist := MinerManagerImpl.GetMinerByID(id, _type, accountdb)
		if mExist == nil {
			Logger.Debugf("TVMExecutor Execute MinerCancelStake Fail(Can not find miner) Source %s", transaction.Source.Hex())
			return
		}
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)
		snapshot := accountdb.Snapshot()
		if MinerManagerImpl.CancelStake(transaction.Source[:], mExist, value, accountdb, height) &&
			MinerManagerImpl.ReduceStake(mExist.ID, mExist, value, accountdb, height) {
			success = true
			Logger.Debugf("TVMExecutor Execute MinerCancelStake success Source %s", transaction.Source.Hex())
		} else {
			Logger.Debugf("TVMExecutor Execute MinerCancelStake Fail(CancelStake or ReduceStake error) Source %s", transaction.Source.Hex())
			accountdb.RevertToSnapshot(snapshot)
		}
	}
	return
}

func (executor *TVMExecutor) executeMinerAbortTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool) {
	success = false

	gasFee := new(types.BigInt).Mul(transaction.GasPrice.Value(), gasUsed.Value())
	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), gasFee) {
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)
		if transaction.Data != nil {
			success = MinerManagerImpl.abortMiner(transaction.Source[:], transaction.Data[0], height, accountdb)
		}
	} else {
		Logger.Debugf("TVMExecutor Execute MinerAbort Fail(Balance Not Enough) Source:%s Height:%d ", transaction.Source.Hex(), height)
	}
	Logger.Debugf("TVMExecutor Execute MinerAbort Tx %s,Source:%s, Success:%t", transaction.Hash.Hex(), transaction.Source.Hex(), success)
	return success
}

func (executor *TVMExecutor) executeMinerRefundTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, gasUsed *types.BigInt) (success bool) {
	success = false

	gasFee := new(types.BigInt).Mul(transaction.GasPrice.Value(), gasUsed.Value())
	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), gasFee) {
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)
		var _type, id, _ = MinerManagerImpl.Transaction2MinerParams(transaction)
		mexist := MinerManagerImpl.GetMinerByID(id, _type, accountdb)
		if mexist != nil {
			snapShot := accountdb.Snapshot()
			defer func() {
				if !success {
					accountdb.RevertToSnapshot(snapShot)
				}
			}()
			if mexist.Type == types.MinerTypeHeavy {
				latestCancelPledgeHeight := MinerManagerImpl.GetLatestCancelStakeHeight(transaction.Source[:], mexist, accountdb)
				if height > latestCancelPledgeHeight+10 || (mexist.Status == types.MinerStatusAbort && height > mexist.AbortHeight+10) {
					value, ok := MinerManagerImpl.RefundStake(transaction.Source.Bytes(), mexist, accountdb)
					if !ok {
						success = false
						return
					}
					amount := new(big.Int).SetUint64(value)
					accountdb.AddBalance(*transaction.Source, amount)
					Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Success %s", transaction.Source.Hex())
					success = true
				} else {
					Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Fail(Refund height less than abortHeight+10) Hash%s", transaction.Source.Hex())
				}
			} else if mexist.Type == types.MinerTypeLight {
				value, ok := MinerManagerImpl.RefundStake(transaction.Source.Bytes(), mexist, accountdb)
				if !ok {
					success = false
					return
				}
				amount := new(big.Int).SetUint64(value)
				accountdb.AddBalance(*transaction.Source, amount)
				Logger.Debugf("TVMExecutor Execute MinerRefund Light Success %s,Type:%s", transaction.Source.Hex())
				success = true
			} else {
				Logger.Debugf("TVMExecutor Execute MinerRefund Fail(No such miner type) %s", transaction.Source.Hex())
				return
			}
		} else {
			Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Not Exist Or Not Abort) %s", transaction.Source.Hex())
		}
	} else {
		Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Balance Not Enough) Hash:%s,Source:%s", transaction.Hash.Hex(), transaction.Source.Hex())
		return success
	}
	return success
}

func createContract(accountdb *account.AccountDB, transaction *types.Transaction) (common.Address, *types.TransactionError) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(transaction.Nonce))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, types.NewTransactionError(types.TxErrorCodeContractAddressConflict, "contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)
	return contractAddr, nil
}

// intrinsicGas means transaction consumption intrinsic gas
func intrinsicGas(transaction *types.Transaction) (gasUsed *types.BigInt, err *types.TransactionError) {
	gas := uint64(float32(len(transaction.Data)+len(transaction.ExtraData)) * CodeBytePrice)
	gas = TransactionGasCost + gas
	gasBig := types.NewBigInt(gas)
	if transaction.GasLimit.Cmp(gasBig.Value()) < 0 {
		return nil, types.TxErrorDeployGasNotEnough
	}
	return types.NewBigInt(gas), nil
}

func canTransfer(db vm.AccountDB, addr common.Address, amount *big.Int, gasFee *big.Int) bool {
	totalAmount := new(big.Int).Add(amount, gasFee)
	return db.GetBalance(addr).Cmp(totalAmount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {

	// Escape if amount is zero
	if amount.Sign() == 0 {
		return
	}
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
