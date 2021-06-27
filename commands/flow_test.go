package commands

import (
	"crypto/rand"
	"fmt"
	"solanoid/commands/executor"
	"solanoid/models/nebula"
	"testing"

	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/types"
)

/*
 * Test logical steps
 *
 * 1. Deploy Nebula
 * 2. Init Nebula
 * 3. Deploy Port
 * 4. Subscribe Port to Nebula
 * 5. Call mocked attach data.
 *
 * Goals:
 * 1. Validate minting flow.
 * 2. Validate oracle multisig. (with various bft*)
 * 3. Validate double spend on attach
 * 4. Validate the atomic call: nebula.send_value_to_subs() -> nebula.attach()
 */
 func TestNebulaSendValueToIBPortSubscriber (t *testing.T) {
	var err error

	deployer, err := NewOperatingAddress(t, "../private-keys/test_deployer-pk-deployer.json", nil)
	ValidateError(t, err)

	gravityProgram, err := NewOperatingAddress(t, "../private-keys/test_only-gravity-program.json", nil)
	ValidateError(t, err)

	nebulaProgram, err := NewOperatingAddress(t, "../private-keys/test_only-nebula-program.json", nil)
	ValidateError(t, err)

	ibportProgram, err := NewOperatingAddress(t, "../private-keys/test_only_ibport-program.json", &OperatingAddressBuilderOptions{
		WithPDASeeds: []byte("ibport"),
	})
	ValidateError(t, err)

	const BFT = 3

	// WrappedFaucet(t, deployer.PKPath, "", 10)

	// waitTransactionConfirmations()

	// WrappedFaucet(t, , "", 10)

	// TransfconsulsList.List[0].Account)

	consulsList, err := GenerateConsuls(t, "../private-keys/_test_consul_prefix_", BFT)
	ValidateError(t, err)

	operatingConsul := consulsList.List[0]
	// WrappedFaucet(t, deployer.PKPath, operatingConsul.PublicKey.ToBase58(), 10)

	for i, consul := range append(consulsList.List, *deployer) {
		if i == BFT {
			WrappedFaucet(t, deployer.PKPath, "", 10)
		}

		WrappedFaucet(t, deployer.PKPath, consul.PublicKey.ToBase58(), 10)
	}

	RPCEndpoint, _ := InferSystemDefinedRPC()

	tokenDeployResult, err := CreateToken(deployer.PKPath)
	ValidateError(t, err)

	tokenProgramAddress := tokenDeployResult.Token.ToBase58()

	// deployerTokenAccount, err := CreateTokenAccount(deployer.PKPath, tokenProgramAddress)
	// ValidateError(t, err)

	waitTransactionConfirmations()


	deployerTokenAccount, err := CreateTokenAccount(deployer.PKPath, tokenProgramAddress)
	ValidateError(t, err)

	gravityDataAccount, err := GenerateNewAccount(deployer.PrivateKey, GravityContractAllocation, gravityProgram.PublicKey.ToBase58(), RPCEndpoint)
	ValidateError(t, err)

	gravityMultisigAccount, err := GenerateNewAccount(deployer.PrivateKey, MultisigAllocation, gravityProgram.PublicKey.ToBase58(), RPCEndpoint)
	ValidateError(t, err)

	nebulaDataAccount, err := GenerateNewAccount(deployer.PrivateKey, NebulaAllocation, nebulaProgram.PublicKey.ToBase58(), RPCEndpoint)
	ValidateError(t, err)

	nebulaMultisigAccount, err := GenerateNewAccount(deployer.PrivateKey, MultisigAllocation, nebulaProgram.PublicKey.ToBase58(), RPCEndpoint)
	ValidateError(t, err)

	ibportDataAccount, err := GenerateNewAccount(deployer.PrivateKey, IBPortAllocation, ibportProgram.PublicKey.ToBase58(), RPCEndpoint)
	ValidateError(t, err)


	ParallelExecution(
		[]func() {
			func() {
				_, err = DeploySolanaProgram(t, "ibport", ibportProgram.PKPath, consulsList.List[0].PKPath, "../binaries/ibport.so")
				ValidateError(t, err)
			},
			func() {
				_, err = DeploySolanaProgram(t, "gravity", gravityProgram.PKPath, consulsList.List[1].PKPath, "../binaries/gravity.so")
				ValidateError(t, err)
			},
			func() {
				_, err = DeploySolanaProgram(t, "nebula", nebulaProgram.PKPath, consulsList.List[2].PKPath, "../binaries/nebula.so")
				ValidateError(t, err)
			},
		},
	)

	waitTransactionConfirmations()

	err = AuthorizeToken(t, deployer.PKPath, tokenProgramAddress, "mint", ibportProgram.PDA.ToBase58())
	ValidateError(t, err)
	t.Log("Authorizing ib port to allow minting")

	waitTransactionConfirmations()
	
	gravityBuilder := executor.GravityInstructionBuilder{}
	gravityExecutor, err := InitGenericExecutor(
		deployer.PrivateKey,
		gravityProgram.PublicKey.ToBase58(),
		gravityDataAccount.Account.PublicKey.ToBase58(),
		gravityMultisigAccount.Account.PublicKey.ToBase58(),
		RPCEndpoint,
		common.PublicKeyFromString(""),
	)
	
	nebulaBuilder := executor.NebulaInstructionBuilder{}
	nebulaExecutor, err := InitGenericExecutor(
		deployer.PrivateKey,
		nebulaProgram.PublicKey.ToBase58(),
		nebulaDataAccount.Account.PublicKey.ToBase58(),
		nebulaMultisigAccount.Account.PublicKey.ToBase58(),
		RPCEndpoint,
		gravityDataAccount.Account.PublicKey,
	)
	ValidateError(t, err)

	ibportBuilder := executor.IBPortInstructionBuilder{}
	ibportExecutor, err := InitGenericExecutor(
		deployer.PrivateKey,
		ibportProgram.PublicKey.ToBase58(),
		ibportDataAccount.Account.PublicKey.ToBase58(),
		"",
		RPCEndpoint,
		common.PublicKeyFromString(""),
	)
	ValidateError(t, err)

	oracles := consulsList.ConcatConsuls()

	waitTransactionConfirmations()

	ParallelExecution(
		[]func() {
			func() {
				gravityInitResponse, err := gravityExecutor.BuildAndInvoke(
					gravityBuilder.Init(BFT, 1, oracles),
				)
				fmt.Printf("Gravity Init: %v \n", gravityInitResponse.TxSignature)
				ValidateError(t, err)
			},
			func() {
				// (2)
				nebulaInitResponse, err := nebulaExecutor.BuildAndInvoke(
					nebulaBuilder.Init(BFT, nebula.Bytes, gravityDataAccount.Account.PublicKey, oracles),
				)
				ValidateError(t, err)
				fmt.Printf("Nebula Init: %v \n", nebulaInitResponse.TxSignature)
			},
			func() {
				ibportInitResult, err := ibportExecutor.BuildAndInvoke(
					ibportBuilder.Init(nebulaProgram.PublicKey, common.TokenProgramID),
				)

				fmt.Printf("IB Port Init: %v \n", ibportInitResult.TxSignature)
				ValidateError(t, err)
			},
		},
	)

	waitTransactionConfirmations()

	fmt.Println("IB Port Program is being subscribed to Nebula")

	var subID [16]byte
    rand.Read(subID[:])
	
	fmt.Printf("subID: %v \n", subID)

	// (4)
	nebulaSubscribePortResponse, err := nebulaExecutor.BuildAndInvoke(
		nebulaBuilder.Subscribe(ibportProgram.PDA, 1, 1, subID),
	)
	ValidateError(t, err)

	fmt.Printf("Nebula Subscribe: %v \n", nebulaSubscribePortResponse.TxSignature)
	fmt.Println("Now checking for valid double spend prevent")

	waitTransactionConfirmations()
	// waitTransactionConfirmations()

	_, err = nebulaExecutor.BuildAndInvoke(
		nebulaBuilder.Subscribe(ibportProgram.PublicKey, 1, 1, subID),
	)
	ValidateErrorExistence(t, err)

	fmt.Printf("Nebula Subscribe with the same subID must have failed: %v \n", err.Error())

	// WrappedFaucet(t, deployer.PKPath, ibportProgram.PublicKey.ToBase58(), 10)

	waitTransactionConfirmations()
	// waitTransactionConfirmations()

	fmt.Println("Testing SendValueToSubs call  from one of the consuls")

	swapId := make([]byte, 16)
    rand.Read(swapId)

	t.Logf("Token Swap Id:  %v \n", swapId)

	attachedAmount := float64(227)

	t.Logf("227 - Float As  Bytes: %v \n", executor.Float64ToBytes(attachedAmount))

	var dataHashForAttach [64]byte
	copy(dataHashForAttach[:], executor.BuildCrossChainMintByteVector(swapId, common.PublicKeyFromString(deployerTokenAccount), attachedAmount))

	fmt.Printf("dataHashForAttach: %v \n", dataHashForAttach)

	nebulaExecutor.SetDeployerPK(operatingConsul.Account)
	nebulaExecutor.SetAdditionalMeta([]types.AccountMeta{
		{ PubKey: common.TokenProgramID, IsWritable: false, IsSigner: false },
		{ PubKey: ibportProgram.PublicKey, IsWritable: false, IsSigner: false },
		{ PubKey: ibportDataAccount.Account.PublicKey, IsWritable: true, IsSigner: false },
		{ PubKey: common.PublicKeyFromString(tokenProgramAddress), IsWritable: true, IsSigner: false },
		{ PubKey: common.PublicKeyFromString(deployerTokenAccount), IsWritable: true, IsSigner: false },
		{ PubKey: ibportProgram.PDA, IsWritable: false, IsSigner: false },
	})

	waitTransactionConfirmations()

	nebulaAttachResponse, err := nebulaExecutor.BuildAndInvoke(
		nebulaBuilder.SendValueToSubs(dataHashForAttach, nebula.Bytes, 1, subID),
	)
	ValidateError(t, err)

	fmt.Printf("Nebula SendValueToSubs Call:  %v \n", nebulaAttachResponse.TxSignature)

	waitTransactionConfirmations()

	nebulaExecutor.EraseAdditionalMeta()
	nebulaExecutor.SetAdditionalSigners(consulsList.ToBftSigners())
	nebulaExecutor.SetDeployerPK(deployer.Account)

	nebulaSendHashValueResponse, err := nebulaExecutor.BuildAndInvoke(
		nebulaBuilder.SendHashValue(dataHashForAttach),
	)
	ValidateError(t, err)

	fmt.Printf("Nebula SendHashValue Call:  %v \n", nebulaSendHashValueResponse.TxSignature)

	// deployerAddress, err := ReadAccountAddress(deployerPrivateKeysPath)
	// ValidateError(t, err)

	// tokenOwnerAddress, err := ReadAccountAddress(tokenOwnerPath)
	// ValidateError(t, err)

	// WrappedFaucet(t, tokenOwnerPath, tokenOwnerAddress, 10)
	// WrappedFaucet(t, deployerPrivateKeysPath, deployerAddress, 10)

	// waitTransactionConfirmations()

	// tokenDeployResult, err := CreateToken(tokenOwnerPath)
	// ValidateError(t, err)

	// tokenProgramAddress := tokenDeployResult.Token.ToBase58()

	// deployerTokenAccount, err := CreateTokenAccount(deployerPrivateKeysPath, tokenProgramAddress)
	// ValidateError(t, err)
	
	// ibportAddressPubkey, ibPortPDA, err := CreatePersistentAccountWithPDA(ibportProgramPath, true, [][]byte{[]byte("ibport")})
	// if err != nil {
	// 	fmt.Printf("PDA error: %v", err)
	// 	t.FailNow()
	// }
	// ibportAddress := ibportAddressPubkey.ToBase58()

	// fmt.Printf("token  program address: %s\n", tokenProgramAddress)

	// t.Logf("tokenProgramAddress: %v", tokenProgramAddress)
	// t.Logf("deployerAddress: %v", deployerAddress)
	// t.Logf("tokenOwnerAddress: %v", tokenOwnerAddress)
	// t.Logf("ibportAddress: %v", ibportAddress)
	// t.Logf("ibPortPDA: %v", ibPortPDA.ToBase58())
	// t.Logf("deployerTokenAccount: %v", deployerTokenAccount)

	// deployerPrivateKey, err := ReadPKFromPath(t, deployerPrivateKeysPath)
	// ValidateError(t, err)
	
}