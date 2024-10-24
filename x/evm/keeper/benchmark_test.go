package keeper_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/sei-protocol/sei-chain/app"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	testkeeper "github.com/sei-protocol/sei-chain/testutil/keeper"
	"github.com/sei-protocol/sei-chain/x/evm/ante"
	evmkeeper "github.com/sei-protocol/sei-chain/x/evm/keeper"
	"github.com/sei-protocol/sei-chain/x/evm/types"
	testfixtures "github.com/sei-protocol/sei-chain/x/evm/types/testfixtures"
)

type KeeperBenchmarkTestSuite struct {
	EvmDenom  string
	App       *app.App
	Ctx       sdk.Context
	EvmKeeper *evmkeeper.Keeper
	ChainID   *big.Int

	MsgServer      types.MsgServer
	EvmQueryClient types.QueryClient
	Signer         ethtypes.Signer

	Key     *ecdsa.PrivateKey
	Address common.Address
}

func (suite *KeeperBenchmarkTestSuite) SetupTest(b *testing.B) {
	suite.EvmDenom = "usei"

	suite.App = app.Setup(false, false)
	suite.Ctx = suite.App.BaseApp.NewContext(false, tmtypes.Header{Height: 1, ChainID: "sei-test", Time: time.Now().UTC()})

	k := suite.App.EvmKeeper
	k.InitGenesis(suite.Ctx, *types.DefaultGenesis())
	suite.App.EvmKeeper = k
	suite.MsgServer = evmkeeper.NewMsgServerImpl(&suite.App.EvmKeeper)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, evmkeeper.NewQuerier(&suite.App.EvmKeeper))
	suite.EvmQueryClient = types.NewQueryClient(queryHelper)

	suite.ChainID = suite.App.EvmKeeper.ChainID(suite.Ctx)
	suite.Signer = ethtypes.LatestSignerForChainID(suite.ChainID)

	privKey := testkeeper.MockPrivateKey()
	_, suite.Address = testkeeper.PrivateKeyToAddresses(privKey)
	testPrivHex := hex.EncodeToString(privKey.Bytes())
	suite.Key, _ = crypto.HexToECDSA(testPrivHex)

	amt := sdk.NewCoins(sdk.NewCoin("usei", sdk.NewInt(100000000)))
	suite.App.BankKeeper.MintCoins(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("usei", sdk.NewInt(100000000))))
	suite.App.BankKeeper.SendCoinsFromModuleToAccount(suite.Ctx, types.ModuleName, suite.Address[:], amt)
}

func (suite *KeeperBenchmarkTestSuite) Commit(t require.TestingT) {
	jumpTime := time.Second * 0
	_, err := suite.App.FinalizeBlock(context.Background(),
		&abci.RequestFinalizeBlock{
			Height: suite.Ctx.BlockHeight(),
			Time:   suite.Ctx.BlockTime(),
		},
	)
	require.NoError(t, err)
	_, err = suite.App.Commit(context.Background())
	require.NoError(t, err)
	newBlockTime := suite.Ctx.BlockTime().Add(jumpTime)
	header := suite.Ctx.BlockHeader()
	header.Time = newBlockTime
	header.Height++

	// update ctx
	suite.Ctx = suite.App.NewUncachedContext(false, header)

	// update query client (new Ctx height)
	queryHelper := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, evmkeeper.NewQuerier(&suite.App.EvmKeeper))
	suite.EvmQueryClient = types.NewQueryClient(queryHelper)
}

// DeployTestContract deploy a test erc20 contract and returns the contract address
func (suite *KeeperBenchmarkTestSuite) DeployTestContract(
	t require.TestingT,
	owner common.Address,
	supply *big.Int,
	enableFeemarket bool,
) common.Address {
	ctorArgs, err := testfixtures.ERC20Contract.ABI.Pack("", owner, supply)
	require.NoError(t, err)
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
	data := append(testfixtures.ERC20Contract.Bin, ctorArgs...) //nolint: gocritic

	// NewMsgEVMTransaction
	var erc20DeployTx *types.MsgEVMTransaction
	// if enableFeemarket {
	// 	erc20DeployTx = types.NewTxContract(
	// 		suite.ChainID,
	// 		nonce,
	// 		nil,     // amount
	// 		10000, // gasLimit
	// 		nil,     // gasPrice
	// 		suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx),
	// 		big.NewInt(1),
	// 		data,                   // input
	// 		&ethtypes.AccessList{}, // accesses
	// 	)
	// } else {
	erc20DeployTx = types.NewTxContract(
		suite.ChainID,
		nonce,
		nil,     // amount
		2000000, // gasLimit
		nil,     // gasPrice
		nil, nil,
		data, // input
		nil,  // accesses
	)
	// }

	tx, _ := erc20DeployTx.AsTransaction()
	signedTx, err := ethtypes.SignTx(tx, suite.Signer, suite.Key)
	require.NoError(t, err)

	msg := types.NewTxWithData(signedTx)
	err = ante.Preprocess(
		suite.Ctx,
		msg,
	)
	require.NoError(t, err)

	rsp, err := suite.MsgServer.EVMTransaction(
		sdk.WrapSDKContext(suite.Ctx),
		msg,
	)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.Address, nonce)
}

// deployTestMessageCall deploy a test erc20 contract and returns the contract address
func (suite *KeeperBenchmarkTestSuite) deployTestMessageCall(b *testing.B) common.Address {
	data := testfixtures.TestMessageCall.Bin

	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
	contractDeployTx := types.NewTxContract(
		suite.ChainID,
		nonce,
		nil,     // amount
		2000000, // gasLimit
		nil,     // gasPrice
		nil, nil,
		data, // input
		nil,  // accesses
	)

	tx, _ := contractDeployTx.AsTransaction()
	signedTx, err := ethtypes.SignTx(tx, suite.Signer, suite.Key)
	require.NoError(b, err)

	msg := types.NewTxWithData(signedTx)
	err = ante.Preprocess(
		suite.Ctx,
		msg,
	)
	require.NoError(b, err)

	rsp, err := suite.MsgServer.EVMTransaction(
		sdk.WrapSDKContext(suite.Ctx),
		msg,
	)
	require.NoError(b, err)
	require.Empty(b, rsp.VmError)
	return crypto.CreateAddress(suite.Address, nonce)
}

func setupContract(b *testing.B) (*KeeperBenchmarkTestSuite, common.Address) {
	suite := KeeperBenchmarkTestSuite{}
	suite.SetupTest(b)

	amt := sdk.Coins{newDefaultCoinInt64(1000 * 1000000)}
	err := suite.App.BankKeeper.MintCoins(suite.Ctx, types.ModuleName, amt)
	require.NoError(b, err)
	err = suite.App.BankKeeper.SendCoinsFromModuleToAccount(suite.Ctx, types.ModuleName, suite.Address.Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.DeployTestContract(b, suite.Address, big.NewInt(1000*1000000), false)
	suite.Commit(b)

	return &suite, contractAddr
}

func setupTestMessageCall(b *testing.B) (*KeeperBenchmarkTestSuite, common.Address) {
	suite := KeeperBenchmarkTestSuite{}
	suite.SetupTest(b)

	amt := sdk.Coins{newDefaultCoinInt64(1000 * 1000000)}
	err := suite.App.BankKeeper.MintCoins(suite.Ctx, types.ModuleName, amt)
	require.NoError(b, err)
	err = suite.App.BankKeeper.SendCoinsFromModuleToAccount(suite.Ctx, types.ModuleName, suite.Address.Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.deployTestMessageCall(b)
	suite.Commit(b)

	return &suite, contractAddr
}

type TxBuilder func(suite *KeeperBenchmarkTestSuite, contract common.Address) *types.MsgEVMTransaction

func doBenchmark(b *testing.B, txBuilder TxBuilder) {
	suite, contractAddr := setupContract(b)

	msg := txBuilder(suite, contractAddr)

	tx, _ := msg.AsTransaction()
	signedTx, err := ethtypes.SignTx(tx, suite.Signer, suite.Key)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.Ctx.CacheContext()

		msg := types.NewTxWithData(signedTx)
		err = ante.Preprocess(
			suite.Ctx,
			msg,
		)
		require.NoError(b, err)

		rsp, err := suite.MsgServer.EVMTransaction(
			sdk.WrapSDKContext(ctx),
			msg,
		)
		require.NoError(b, err)
		require.False(b, rsp.VmError != "")
	}
}

func BenchmarkTokenTransfer(b *testing.B) {
	doBenchmark(b, func(suite *KeeperBenchmarkTestSuite, contract common.Address) *types.MsgEVMTransaction {
		input, err := testfixtures.ERC20Contract.ABI.Pack("transfer", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
		return types.NewTx(suite.ChainID, nonce, &contract, big.NewInt(0), 410000, big.NewInt(1), nil, nil, input, nil)
	})
}

func BenchmarkEmitLogs(b *testing.B) {
	doBenchmark(b, func(suite *KeeperBenchmarkTestSuite, contract common.Address) *types.MsgEVMTransaction {
		input, err := testfixtures.ERC20Contract.ABI.Pack("benchmarkLogs", big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
		return types.NewTx(suite.ChainID, nonce, &contract, big.NewInt(0), 4100000, big.NewInt(1), nil, nil, input, nil)
	})
}

func BenchmarkTokenTransferFrom(b *testing.B) {
	doBenchmark(b, func(suite *KeeperBenchmarkTestSuite, contract common.Address) *types.MsgEVMTransaction {
		input, err := testfixtures.ERC20Contract.ABI.Pack("transferFrom", suite.Address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(0))
		require.NoError(b, err)
		nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
		return types.NewTx(suite.App.EvmKeeper.ChainID(
			suite.Ctx,
		), nonce, &contract, big.NewInt(0), 410000, big.NewInt(1), nil, nil, input, nil)
	})
}

func BenchmarkTokenMint(b *testing.B) {
	doBenchmark(b, func(suite *KeeperBenchmarkTestSuite, contract common.Address) *types.MsgEVMTransaction {
		input, err := testfixtures.ERC20Contract.ABI.Pack("mint", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
		return types.NewTx(suite.ChainID, nonce, &contract, big.NewInt(0), 410000, big.NewInt(1), nil, nil, input, nil)
	})
}

func BenchmarkMessageCall(b *testing.B) {
	suite, contract := setupTestMessageCall(b)

	input, err := testfixtures.TestMessageCall.ABI.Pack("benchmarkMessageCall", big.NewInt(10000))
	require.NoError(b, err)
	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
	msg := types.NewTx(suite.App.EvmKeeper.ChainID(
		suite.Ctx,
	), nonce, &contract, big.NewInt(0), 25000000, big.NewInt(1), nil, nil, input, nil)

	tx, _ := msg.AsTransaction()
	signedTx, err := ethtypes.SignTx(tx, suite.Signer, suite.Key)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.Ctx.CacheContext()

		msg := types.NewTxWithData(signedTx)
		err = ante.Preprocess(
			suite.Ctx,
			msg,
		)
		require.NoError(b, err)

		rsp, err := suite.MsgServer.EVMTransaction(
			sdk.WrapSDKContext(ctx),
			msg,
		)
		require.NoError(b, err)
		require.False(b, rsp.VmError != "")
	}
}

func newDefaultCoinInt64(amount int64) sdk.Coin {
	return sdk.NewInt64Coin("usei", amount)
}
