package logical

import (
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware"
	"github.com/zvchain/zvchain/middleware/time"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/taslog"
	"gopkg.in/fatih/set.v0"
	"strings"
	"testing"
	time2 "time"
)

func GenTestBH(param string, value ...interface{}) types.BlockHeader {
	bh := types.BlockHeader{}
	bh.Elapsed = 1
	switch param {
	case "Hash":
		bh.Hash = common.HexToHash("0x01")
	case "Height":
		bh.Height = 10
		bh.Hash = bh.GenHash()
	case "PreHash":
		bh.PreHash = common.HexToHash("0x02")
		bh.Hash = bh.GenHash()
	case "Elapsed":
		bh.Elapsed = 100
		bh.Hash = bh.GenHash()
	case "ProveValue":
		bh.ProveValue = []byte{0, 1, 2}
		bh.Hash = bh.GenHash()
	case "TotalQN":
		bh.TotalQN = 10
		bh.Hash = bh.GenHash()
	case "CurTime":
		bh.CurTime = time.TimeToTimeStamp(time2.Now())
		bh.Hash = bh.GenHash()
	case "Castor":
		bh.Castor = []byte{0, 1}
		bh.Hash = bh.GenHash()
	case "Group":
		bh.Group = common.HexToHash("0x03")
		bh.Hash = bh.GenHash()
	case "Signature":
		bh.Signature = []byte{23, 22}
		bh.Hash = bh.GenHash()
	case "Nonce":
		bh.Nonce = 12
		bh.Hash = bh.GenHash()
	case "TxTree":
		bh.TxTree = common.HexToHash("0x04")
		bh.Hash = bh.GenHash()
	case "ReceiptTree":
		bh.ReceiptTree = common.HexToHash("0x05")
		bh.Hash = bh.GenHash()
	case "StateTree":
		bh.StateTree = common.HexToHash("0x06")
		bh.Hash = bh.GenHash()
	case "ExtraData":
		bh.ExtraData = []byte{4, 22}
		bh.Hash = bh.GenHash()
	case "Random":
		bh.Random = []byte{4, 22, 145}
		bh.Hash = bh.GenHash()
	case "GasFee":
		bh.GasFee = 123
		bh.Hash = bh.GenHash()
	case "Castor=getMinerId":
		bh.Castor = common.FromHex("0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676")
		bh.Hash = bh.GenHash()
	case "bh.Elapsed<=0":
		bh.Elapsed = -2
		bh.Height = 10
		bh.Hash = bh.GenHash()
	case "p.ts.Since(bh.CurTime)<-1":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) + 2
		bh.Hash = bh.GenHash()
	case "block-exists":
		bh.GasFee = 10
		bh.Hash = bh.GenHash()
	case "pre-block-not-exists":
		bh.PreHash = common.HexToHash("0x01")
		bh.Hash = bh.GenHash()
	case "already-cast":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 1
		bh.PreHash = common.HexToHash("0x1234")
		bh.Height = 1
		bh.Hash = bh.GenHash()
	case "already-sign":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 3
		bh.PreHash = common.HexToHash("0x02")
		bh.Height = 2
		bh.Hash = bh.GenHash()
	case "cast-illegal":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 3
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 3
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.ProveValue = common.FromHex("0x03556a119b69e52a6c8f676213e2184c588bc9731ec0ab1ed32a91a9a22155cdeb001fa9a2fd33c8660483f267050f0e72072658f16d485a1586fca736a50a423cbbb181870219af0c2c4fdbbb89832730")
		bh.Hash = bh.GenHash()
	case "slot-is-nil":
		bh.CurTime = time.TimeToTimeStamp(time2.Now()) - 3
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 3
		bh.Hash = bh.GenHash()
	case "not-in-verify-group":
		bh.CurTime = time.TimeToTimeStamp(time2.Now())
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 3
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	case "sender-not-in-verify-group":
		bh.CurTime = time.TimeToTimeStamp(time2.Now())
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 4
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	case "receive-before-proposal":
		bh.PreHash = common.HexToHash("0x03")
		bh.Height = 4
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	case "already-sign-bigger-weight":
		bh.PreHash = common.HexToHash("0x02")
		bh.Height = 6
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	case "height-casted":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 7
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	case "has-signed":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 8
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	case "to51":
		bh.PreHash = common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")
		bh.Height = 9
		bh.Castor = common.Hex2Bytes("0000000100000000000000000000000000000000000000000000000000000000")
		bh.Hash = bh.GenHash()
	}

	return bh
}

func GenTestBHHash(param string) common.Hash {
	bh := GenTestBH(param)
	return bh.Hash
}

func EmptyBHHash() common.Hash {
	bh := types.BlockHeader{}
	bh.Elapsed = 1
	return bh.GenHash()
}

var emptyBHHash = EmptyBHHash()

func NewProcess2() *Processor {
	common.InitConf("./tas_config_test.ini")
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	err := middleware.InitMiddleware()
	if err != nil {
		panic(err)
	}
	err = core.InitCore(NewConsensusHelper4Test(groupsig.ID{}), nil)
	core.GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})

	process := &Processor{}
	sk := common.HexToSecKey(getAccount().Sk)
	minerInfo, _ := model.NewSelfMinerDO(sk)
	InitConsensus()
	process.Init(minerInfo, common.GlobalConf)
	//hijack some external interface to avoid error
	process.MainChain = &chain4Test{core.BlockChainImpl}
	process.NetServer = &networkServer4Test{process.NetServer}
	return process
}

func TestProcessor_OnMessageCast(t *testing.T) {
	_ = initContext4Test()
	defer clear()

	pt := NewProcessorTest()
	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	//txs := make([]*types.Transaction, 0)
	type args struct {
		msg *model.ConsensusCastMessage
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "Height Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Height"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Height"), emptyBHHash, GenTestBHHash("Height")),
		},
		{
			name: "PreHash Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("PreHash"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("PreHash"), emptyBHHash, GenTestBHHash("PreHash")),
		},
		{
			name: "Elapsed Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Elapsed"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Elapsed"), emptyBHHash, GenTestBHHash("Elapsed")),
		},
		{
			name: "ProveValue Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("ProveValue"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("ProveValue"), emptyBHHash, GenTestBHHash("ProveValue")),
		},
		{
			name: "TotalQN Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("TotalQN"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("TotalQN"), emptyBHHash, GenTestBHHash("TotalQN")),
		},
		{
			name: "CurTime Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("CurTime"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("CurTime"), emptyBHHash, GenTestBHHash("CurTime")),
		},
		{
			name: "Castor Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Castor"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Castor"), emptyBHHash, GenTestBHHash("Castor")),
		},
		{
			name: "Group Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Group"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Group"), emptyBHHash, GenTestBHHash("Group")),
		},
		{
			name: "Signature Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Signature"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Signature"), emptyBHHash, GenTestBHHash("Signature")),
		},
		{
			name: "Nonce Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Nonce"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Nonce"), emptyBHHash, GenTestBHHash("Nonce")),
		},
		{
			name: "TxTree Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("TxTree"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("TxTree"), emptyBHHash, GenTestBHHash("TxTree")),
		},
		{
			name: "ReceiptTree Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("ReceiptTree"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("ReceiptTree"), emptyBHHash, GenTestBHHash("ReceiptTree")),
		},
		{
			name: "StateTree Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("StateTree"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("StateTree"), emptyBHHash, GenTestBHHash("StateTree")),
		},
		{
			name: "ExtraData Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("ExtraData"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("ExtraData"), emptyBHHash, GenTestBHHash("ExtraData")),
		},
		{
			name: "Random Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Random"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("Random"), emptyBHHash, GenTestBHHash("Random")),
		},
		{
			name: "GasFee Check",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("GasFee"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("msg genHash %v diff from si.DataHash %v || bh.Hash %v", GenTestBHHash("GasFee"), emptyBHHash, GenTestBHHash("GasFee")),
		},
		{
			name: "Castor=getMinerId",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("Castor=getMinerId"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("Castor=getMinerId"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "ignore self message",
		},
		{
			name: "bh.Elapsed<=0",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("bh.Elapsed<=0"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("bh.Elapsed<=0"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: fmt.Sprintf("elapsed error %v", -1),
		},
		{
			name: "p.ts.Since(bh.CurTime)<-1",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("p.ts.Since(bh.CurTime)<-1"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("p.ts.Since(bh.CurTime)<-1"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block too early",
		},
		{
			name: "block-exists",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("block-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("block-exists"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block onchain already",
		},
		{
			name: "pre-block-not-exists",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("pre-block-not-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("pre-block-not-exists"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "parent block did not received",
		},
		{
			name: "already-cast",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("already-cast"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("already-cast"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "the block of this height has been cast",
		},
		{
			name: "already-sign",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("already-sign"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("already-sign"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block signed",
		},
		{
			name: "cast-illegal",
			args: args{
				msg: &model.ConsensusCastMessage{
					BH: GenTestBH("cast-illegal"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("cast-illegal"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "miner can't cast at height",
		},
	}
	p := processorTest
	p.groupReader.cache.Add(common.HexToHash("0x00"), &verifyGroup{memIndex: map[string]int{
		"0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676": 0,
	}, members: []*member{&member{}}})
	// for already-cast
	p.blockContexts.addCastedHeight(1, common.HexToHash("0x1234"))
	// for already-sign
	vcx := &VerifyContext{}
	vcx.castHeight = 2
	vcx.signedBlockHashs = set.New(set.ThreadSafe)
	vcx.signedBlockHashs.Add(GenTestBHHash("already-sign"))
	p.blockContexts.addVctx(vcx)
	// for cast-illegal
	p.minerReader = newMinerPoolReader(p, NewMinerPoolTest(pt.mpk, pt.ids, pt.verifyGroup))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := p.OnMessageCast(tt.args.msg)
			if msg != nil && !strings.Contains(msg.Error(), tt.expected) {
				t.Errorf("wanted {%s}; got {%s}", tt.expected, msg)
			}
		})
	}
}

func TestProcessor_OnMessageVerify(t *testing.T) {
	_ = initContext4Test()
	defer clear()

	pt := NewProcessorTest()
	processorTest.blockContexts.attachVctx(pt.blockHeader, pt.verifyContext)
	//txs := make([]*types.Transaction, 0)
	type args struct {
		msg *model.ConsensusVerifyMessage
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name: "block-exists",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("block-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block already on chain",
		},
		{
			name: "slot-is-nil",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("slot-is-nil"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "slot is nil",
		},
		{
			name: "Castor=getMinerId",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("Castor=getMinerId"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "ignore self message",
		},
		{
			name: "not-in-verify-group",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("not-in-verify-group"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "don't belong to verifyGroup",
		},
		{
			name: "sender-not-in-verify-group",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("sender-not-in-verify-group"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[8], pt.msk[1]),
					},
				},
			},
			expected: "sender doesn't belong the verifyGroup",
		},
		{
			name: "bh.Elapsed<=0",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("bh.Elapsed<=0"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "elapsed error",
		},
		{
			name: "p.ts.Since(bh.CurTime)<-1",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("p.ts.Since(bh.CurTime)<-1"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "block too early",
		},
		{
			name: "receive-before-proposal",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("receive-before-proposal"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "verify context is nil, cache msg",
		},
		{
			name: "receive-before-proposal",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("receive-before-proposal"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "verify context is nil, cache msg",
		},
		{
			name: "already-sign-bigger-weight",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("already-sign-bigger-weight"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "have signed a higher qn block",
		},
		{
			name: "pre-block-not-exists",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("pre-block-not-exists"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(emptyBHHash, pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "pre not on chain",
		},
		{
			name: "height-casted",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("height-casted"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("height-casted"), pt.ids[1], pt.msk[1]),
					},
				},
			},
			expected: "the block of this height has been cast",
		},
		{
			name: "has-signed",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("has-signed"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("has-signed"), pt.ids[1], pt.msk[1]),
					},
					RandomSign: groupsig.Sign(pt.msk[1], []byte{1}),
				},
			},
			expected: "duplicate message",
		},
		{
			name: "to51",
			args: args{
				msg: &model.ConsensusVerifyMessage{
					BlockHash: GenTestBHHash("to51"),
					BaseSignedMessage: model.BaseSignedMessage{
						SI: model.GenSignData(GenTestBHHash("to51"), pt.ids[1], pt.msk[1]),
					},
					RandomSign: groupsig.Sign(pt.msk[1], []byte{1}),
				},
			},
			expected: "",
		},
	}
	p := processorTest
	p.groupReader.cache.Add(common.HexToHash("0x00"), &verifyGroup{memIndex: map[string]int{
		"0x7310415c8c1ba2b1b074029a9a663ba20e8bba3fa7775d85e003b32b43514676": 1,
	}, members: []*member{&member{}}})
	// for block-exists
	testBH1 := GenTestBH("block-exists")
	p.blockContexts.attachVctx(&testBH1, &VerifyContext{})
	testBH2 := GenTestBH("slot-is-nil")
	p.blockContexts.attachVctx(&testBH2, &VerifyContext{})
	testBH3 := GenTestBH("Castor=getMinerId")
	p.blockContexts.attachVctx(&testBH3, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH3.Hash: {castor: groupsig.DeserializeID(testBH3.Castor)}},
	})
	testBH4 := GenTestBH("not-in-verify-group")
	p.blockContexts.attachVctx(&testBH4, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH4.Hash: {BH: &testBH4, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header: GroupHeaderTest{},
		},
		ts: p.ts,
	})
	testBH5 := GenTestBH("sender-not-in-verify-group")
	p.blockContexts.attachVctx(&testBH5, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH5.Hash: {BH: &testBH5, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 1},
		},
		ts: p.ts,
	})
	testBH6 := GenTestBH("bh.Elapsed<=0")
	p.blockContexts.attachVctx(&testBH6, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH6.Hash: {BH: &testBH6, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 1},
		},
		ts: p.ts,
	})

	testBH7 := GenTestBH("p.ts.Since(bh.CurTime)<-1")
	p.blockContexts.attachVctx(&testBH7, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH7.Hash: {BH: &testBH7, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 1},
		},
		ts: p.ts,
	})

	testBH8 := GenTestBH("already-sign-bigger-weight")
	vctx := &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH8.Hash: {BH: &testBH8, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 1},
		},
		ts:               p.ts,
		signedBlockHashs: set.New(set.ThreadSafe),
		castHeight:       testBH8.Height,
	}
	p.blockContexts.attachVctx(&testBH8, vctx)
	copyTestBH8 := testBH8
	copyTestBH8.TotalQN = 10000
	vctx.markSignedBlock(&copyTestBH8)
	testBH9 := GenTestBH("pre-block-not-exists")
	p.blockContexts.attachVctx(&testBH9, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH9.Hash: {BH: &testBH9, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 1, pt.ids[1].GetHexString(): 1},
		},
		ts:     p.ts,
		prevBH: genBlockHeader(),
	})
	testBH10 := GenTestBH("height-casted")
	p.blockContexts.attachVctx(&testBH10, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH10.Hash: {BH: &testBH10, gSignGenerator: model.NewGroupSignGenerator(2)}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 1, pt.ids[1].GetHexString(): 1},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501")},
	})
	p.blockContexts.recentCasted.Add(testBH10.Height, &castedBlock{height: testBH10.Height, preHash: testBH10.PreHash})
	testBH11 := GenTestBH("has-signed")
	gsg := model.NewGroupSignGenerator(2)
	gsg.AddWitness(pt.ids[1], groupsig.Sign(pt.msk[1], GenTestBHHash("has-signed").Bytes()))
	p.blockContexts.attachVctx(&testBH11, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH11.Hash: {BH: &testBH11, gSignGenerator: gsg}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{{pt.ids[1], pt.mpk[1]}},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 0, pt.ids[1].GetHexString(): 0},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501"), Random: []byte{1}},
	})
	testBH12 := GenTestBH("to51")
	rsg := model.NewGroupSignGenerator(2)
	rsg.AddWitnessForce(pt.ids[8], groupsig.Sign(pt.msk[2], []byte{1}))
	p.blockContexts.attachVctx(&testBH12, &VerifyContext{
		slots: map[common.Hash]*SlotContext{testBH12.Hash: {BH: &testBH12, gSignGenerator: model.NewGroupSignGenerator(1), rSignGenerator: rsg}},
		group: &verifyGroup{
			header:   GroupHeaderTest{},
			members:  []*member{{pt.ids[1], pt.mpk[1]}},
			memIndex: map[string]int{p.GetMinerID().GetHexString(): 0, pt.ids[1].GetHexString(): 0},
		},
		ts:     p.ts,
		prevBH: &types.BlockHeader{Hash: common.HexToHash("0x151c6bde6409e99bc90aae2eded5cec1b7ee6fd2a9f57edb9255c776b4dfe501"), Random: []byte{1}},
	})
	// for cast-illegal
	p.minerReader = newMinerPoolReader(p, NewMinerPoolTest(pt.mpk, pt.ids, pt.verifyGroup))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := p.OnMessageVerify(tt.args.msg)
			if msg != nil && !strings.Contains(msg.Error(), tt.expected) {
				t.Errorf("wanted {%s}; got {%s}", tt.expected, msg)
			}
		})
	}
}

type GroupHeaderTest struct{}

func (GroupHeaderTest) Seed() common.Hash {
	return common.HexToHash("0x00")
}

func (GroupHeaderTest) WorkHeight() uint64 {
	panic("implement me")
}

func (GroupHeaderTest) DismissHeight() uint64 {
	panic("implement me")
}

func (GroupHeaderTest) PublicKey() []byte {
	return []byte{}
}

func (GroupHeaderTest) Threshold() uint32 {
	panic("implement me")
}

func (GroupHeaderTest) GroupHeight() uint64 {
	panic("implement me")
}
