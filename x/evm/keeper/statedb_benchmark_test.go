package keeper_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/sei-protocol/sei-chain/app"
	testkeeper "github.com/sei-protocol/sei-chain/testutil/keeper"
	"github.com/sei-protocol/sei-chain/x/evm/keeper"
	"github.com/sei-protocol/sei-chain/x/evm/state"
	"github.com/stretchr/testify/require"
)

type StatedbBenchmarkTestSuite struct {
	App       *app.App
	EVMKeeper *keeper.Keeper
	Ctx       sdk.Context

	Address common.Address
}

func (suite *StatedbBenchmarkTestSuite) SetupTest(b *testing.B) {
	suite.EVMKeeper, suite.Ctx = testkeeper.MockEVMKeeper()
	suite.App = testkeeper.EVMTestApp
	suite.App.EvmKeeper = *suite.EVMKeeper

	key := testkeeper.MockPrivateKey()
	_, suite.Address = testkeeper.PrivateKeyToAddresses(key)
}

func (suite *StatedbBenchmarkTestSuite) StateDB() *state.DBImpl {
	return state.NewDBImpl(suite.Ctx, &suite.App.EvmKeeper, false)
}

func BenchmarkCreateAccountNew(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		addr := generateAddress()
		b.StartTimer()
		vmdb.CreateAccount(addr)
	}
}

func BenchmarkCreateAccountExisting(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.CreateAccount(suite.Address)
	}
}

func BenchmarkAddBalance(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	amt := big.NewInt(10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.AddBalance(suite.Address, uint256.MustFromBig(amt).ToBig(), tracing.BalanceChangeUnspecified)
	}
}

func BenchmarkSetCode(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	hash := crypto.Keccak256Hash([]byte("code")).Bytes()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.SetCode(suite.Address, hash)
	}
}

func BenchmarkSetState(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	hash := crypto.Keccak256Hash([]byte("topic")).Bytes()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.SetCode(suite.Address, hash)
	}
}

func BenchmarkAddLog(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	topic := crypto.Keccak256Hash([]byte("topic"))
	txHash := crypto.Keccak256Hash([]byte("tx_hash"))
	blockHash := crypto.Keccak256Hash([]byte("block_hash"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.AddLog(&ethtypes.Log{
			Address:     suite.Address,
			Topics:      []common.Hash{topic},
			Data:        []byte("data"),
			BlockNumber: 1,
			TxHash:      txHash,
			TxIndex:     1,
			BlockHash:   blockHash,
			Index:       1,
			Removed:     false,
		})
	}
}

func BenchmarkSnapshot(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		target := vmdb.Snapshot()
		require.Equal(b, i+1, target)
	}

	for i := b.N - 1; i >= 0; i-- {
		require.NotPanics(b, func() {
			vmdb.RevertToSnapshot(i + 1)
		})
	}
}

func BenchmarkSubBalance(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	amt := big.NewInt(10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.SubBalance(suite.Address, uint256.MustFromBig(amt).ToBig(), tracing.BalanceChangeUnspecified)
	}
}

func BenchmarkSetNonce(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.SetNonce(suite.Address, 1)
	}
}

func BenchmarkAddRefund(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vmdb.AddRefund(1)
	}
}

func BenchmarkSuicide(b *testing.B) {
	suite := StatedbBenchmarkTestSuite{}
	suite.SetupTest(b)
	vmdb := suite.StateDB()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		addr := generateAddress()
		vmdb.CreateAccount(addr)
		b.StartTimer()

		vmdb.SelfDestruct(addr)
	}
}

func generateAddress() common.Address {
	key := testkeeper.MockPrivateKey()
	_, addr := testkeeper.PrivateKeyToAddresses(key)

	return addr
}
