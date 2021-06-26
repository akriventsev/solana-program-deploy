package commands

import (
	"crypto/rand"
	"fmt"
	"solanoid/commands/executor"
	"solanoid/models/nebula"
	"sync"
	"time"

	"testing"

	"github.com/mr-tron/base58"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/types"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestGravityContract(t *testing.T) {
	var err, errFailing error

	deployerPrivateKeyPath := "../private-keys/main-deployer.json"
	// deployerPrivateKeyPath := "/Users/shamil/.config/solana/id.json"
	deployerPrivateKey, err := ReadPKFromPath(t, deployerPrivateKeyPath)
	ValidateError(t, err)

	deployerAddress, err := ReadAccountAddress(deployerPrivateKeyPath)
	ValidateError(t, err)

	initialBalance, err := ReadAccountBalance(deployerAddress)
	ValidateError(t, err)

	gravityProgramID, err := DeploySolanaProgram(t, "gravity", "../private-keys/gravity3.json", deployerPrivateKeyPath, "../binaries/gravity.so")
	ValidateError(t, err)

	endpoint, _ := InferSystemDefinedRPC()

	gravityStateAccount, err := GenerateNewAccount(deployerPrivateKey, GravityContractAllocation, gravityProgramID, endpoint)
	ValidateError(t, err)

	gravityMultisigAccount, err := GenerateNewAccount(deployerPrivateKey, MultisigAllocation, gravityProgramID, endpoint)
	ValidateError(t, err)

	bft := uint8(3)
	consulsPKlist := make([]types.Account, bft)

	var consulsKeysList []common.PublicKey

	for i := range consulsPKlist {
		consul := types.NewAccount()
		consulsPKlist[i] = consul

		consulsKeysList = append(consulsKeysList, consul.PublicKey)
	}

	// consulsKeysList := []common.PublicKey {
	// 	common.PublicKeyFromString("EnwGpvfZdCpkjs8jMShjo8evce2LbNfrYvREzdwGh5oc"),
	// 	common.PublicKeyFromString("ESgKDVemBdqDty6WExZ74kV8Re9yepth5tbKcsWTNXC9"),
	// 	common.PublicKeyFromString("5Ng92o7CPPWk5tT2pqrnRMndoD49d51f4QcocgJttGHS"),
	// }

	consuls := make([]byte, 0)
	for _, x := range consulsKeysList {
		consuls = append(consuls, x.Bytes()...)
	}

	// _, err = InitGravity(
	// 	deployerPrivateKey, gravityProgramID,
	// 	gravityStateAccount.Account.PublicKey.ToBase58(),
	// 	gravityMultisigAccount.Account.PublicKey.ToBase58(),
	// 	endpoint,
	// 	consuls,
	// )
	// ValidateError(t, err)
	time.Sleep(time.Second * 20)

	gravityExecutor, err := InitGenericExecutor(
		deployerPrivateKey,
		gravityProgramID,
		gravityStateAccount.Account.PublicKey.ToBase58(),
		gravityMultisigAccount.Account.PublicKey.ToBase58(),
		endpoint,
		common.PublicKeyFromString(gravityProgramID),
	)
	ValidateError(t, err)

	// t.Logf("before - Gravity Consuls Update should fail - program account is not initialized: %v \n", errFailing)

	_, errFailing = gravityExecutor.BuildAndInvoke(executor.UpdateConsulsGravityContractInstruction{
		Instruction: 1,
		Bft:         bft,
		LastRound:   10,
		Consuls:     append(consuls[:], consuls[:]...),
	})
	ValidateErrorExistence(t, errFailing)

	t.Logf("Gravity Consuls Update should fail - program account is not initialized: %v \n", errFailing)

	time.Sleep(time.Second * 20)

	gravityInitResponse, err := gravityExecutor.BuildAndInvoke(executor.InitGravityContractInstruction{
		Instruction: 0,
		Bft:         bft,
		InitRound:   1,
		Consuls:     consuls[:],
	})
	ValidateError(t, err)

	t.Logf("Gravity Init: %v \n", gravityInitResponse.TxSignature)

	time.Sleep(time.Second * 20)

	var signers []executor.GravityBftSigner
	// var additionalMeta []types.AccountMeta

	for _, signer := range consulsPKlist {
		signers = append(signers, *executor.NewGravityBftSigner(base58.Encode(signer.PrivateKey)))
		// additionalMeta = append(additionalMeta, types.AccountMeta{
		// 	PubKey: common.PublicKeyFromString(solana.ClockProgram), IsSigner: false, IsWritable: false
		// })
	}

	gravityExecutor.SetAdditionalSigners(signers)
	// gravityExecutor.SetAdditionalMeta(additionalMeta)
	// nebulaExecutor.SetAdditionalMeta([]types.AccountMeta {
	// 	{ PubKey: common.PublicKeyFromString(solana.ClockProgram), IsSigner: false, IsWritable: false },
	// })

	gravityConsulsUpdateResponse, err := gravityExecutor.BuildAndInvoke(executor.UpdateConsulsGravityContractInstruction{
		Instruction: 1,
		Bft:         bft,
		LastRound:   10,
		Consuls:     consuls,
	})
	ValidateError(t, err)

	t.Logf("Gravity Consuls Update: %v \n", gravityConsulsUpdateResponse.TxSignature)

	time.Sleep(time.Second * 20)
	_, errFailing = gravityExecutor.BuildAndInvoke(executor.UpdateConsulsGravityContractInstruction{
		Instruction: 1,
		Bft:         bft,
		LastRound:   0,
		Consuls:     consuls,
	})
	ValidateErrorExistence(t, errFailing)

	t.Logf("Gravity Consuls Update should fail - invalid last round: %v \n", errFailing)

	aftermathBalance, err := ReadAccountBalance(deployerAddress)
	ValidateError(t, err)

	t.Log("Deploy result in a success")
	t.Logf("Gravity Program ID: %v \n", gravityProgramID)
	t.Logf("Spent: %v SOL \n", initialBalance-aftermathBalance)
}
func TestPDA(t *testing.T) {
	tokenPDA, err := common.CreateProgramAddress([][]byte{[]byte("ibporttheminter"), []byte("ibporttheminter2")}, common.PublicKeyFromString("AgR3ZKBx7Ce7vLDBqX33uZAHELvB8z2Uu3exKDVNmVhU"))
	if err != nil {
		fmt.Printf("PDA error: %v\n", err)
		t.FailNow()
	}
	fmt.Printf("PDA: %s\n", tokenPDA.ToBase58())
	t.FailNow()
}

func waitTransactionConfirmations() {
	time.Sleep(time.Second * 30)
}

func WrappedFaucet(t *testing.T, receiverPath, receiverAddress string, amount uint64) {
	// err := SystemFaucet(t, receiverAddress, amount)
	// if err != nil {
	// 	err = SystemAirdrop(t, receiverPath, amount)
	// 	ValidateError(t, err)
	// }
	t.Logf("Faucet %v SOL to %v \n", receiverAddress, fmt.Sprint(amount))
	err := SystemAirdrop(t, receiverPath, amount)
	ValidateError(t, err)
}


func TestIBPortContract(t *testing.T) {
	var err error
	deployerPrivateKeysPath := "../private-keys/_test_deployer-pk-deployer.json"
	tokenOwnerPath := "../private-keys/_test_only-token-owner.json"
	ibportProgramPath := "../private-keys/_test_only_ibport-owner.json"

	err = CreatePersistedAccount(deployerPrivateKeysPath, true)
	ValidateError(t, err)
	err = CreatePersistedAccount(tokenOwnerPath, true)
	ValidateError(t, err)
	err = CreatePersistedAccount(ibportProgramPath, true)
	ValidateError(t, err)

	deployerAddress, err := ReadAccountAddress(deployerPrivateKeysPath)
	ValidateError(t, err)

	tokenOwnerAddress, err := ReadAccountAddress(tokenOwnerPath)
	ValidateError(t, err)


	// err = SystemFaucet(t, tokenOwnerAddress, 10)
	WrappedFaucet(t, tokenOwnerPath, tokenOwnerAddress, 10)
	ValidateError(t, err)
	WrappedFaucet(t, deployerPrivateKeysPath, deployerAddress, 10)
	ValidateError(t, err)

	tokenDeployResult, err := CreateToken(tokenOwnerPath)
	ValidateError(t, err)

	tokenProgramAddress := tokenDeployResult.Token.ToBase58()

	deployerTokenAccount, err := CreateTokenAccount(deployerPrivateKeysPath, tokenProgramAddress)
	ValidateError(t, err)
	
	ibportAddressPubkey, ibPortPDA, err := CreatePersistentAccountWithPDA(ibportProgramPath, true, [][]byte{[]byte("ibport")})
	if err != nil {
		fmt.Printf("PDA error: %v", err)
		t.FailNow()
	}
	ibportAddress := ibportAddressPubkey.ToBase58()

	fmt.Printf("token  program address: %s\n", tokenProgramAddress)

	t.Logf("tokenProgramAddress: %v", tokenProgramAddress)
	t.Logf("deployerAddress: %v", deployerAddress)
	t.Logf("tokenOwnerAddress: %v", tokenOwnerAddress)
	t.Logf("ibportAddress: %v", ibportAddress)
	t.Logf("ibPortPDA: %v", ibPortPDA.ToBase58())
	t.Logf("deployerTokenAccount: %v", deployerTokenAccount)

	deployerPrivateKey, err := ReadPKFromPath(t, deployerPrivateKeysPath)
	ValidateError(t, err)

	// SystemFaucet(t, deployerAddress, 10)
	// ValidateError(t, err)

	// love this *ucking timeouts
	waitTransactionConfirmations()

	_, err = DeploySolanaProgram(t, "ibport", ibportProgramPath, deployerPrivateKeysPath, "../binaries/ibport.so")
	ValidateError(t, err)

	endpoint, _ := InferSystemDefinedRPC()

	portDataAccount, err := GenerateNewAccount(deployerPrivateKey, IBPortAllocation, ibportAddress, endpoint)
	ValidateError(t, err)

	ibportExecutor, err := InitGenericExecutor(
		deployerPrivateKey,
		ibportAddress,
		portDataAccount.Account.PublicKey.ToBase58(),
		"",
		endpoint,
		common.PublicKeyFromString(ibportAddress),
	)
	ValidateError(t, err)

	instructionBuilder := executor.NewIBPortInstructionBuilder()

	waitTransactionConfirmations()
	ibportInitResult, err := ibportExecutor.BuildAndInvoke(
		instructionBuilder.Init(common.PublicKeyFromBytes(make([]byte, 32)), common.TokenProgramID),
	)
	ValidateError(t, err)
	t.Logf("IBPort Init: %v \n", ibportInitResult.TxSignature)

	ibportExecutor.SetAdditionalMeta([]types.AccountMeta{
		{ PubKey: common.TokenProgramID, IsWritable: false, IsSigner: false },
		{ PubKey: common.PublicKeyFromString(tokenProgramAddress), IsWritable: true, IsSigner: false },
		{ PubKey: common.PublicKeyFromString(deployerTokenAccount), IsWritable: true, IsSigner: false },
		{ PubKey: ibPortPDA, IsWritable: false, IsSigner: false },
	})

	burnAmount := float64(10)

	// mint some tokens for deployer
	err = MintToken(tokenOwnerPath, tokenProgramAddress, burnAmount, deployerTokenAccount)
	ValidateError(t, err)
	t.Log("Minted  some tokens")

	waitTransactionConfirmations()

	// delegate amount to port BINARY for burning and request creation
	err = DelegateSPLTokenAmount(deployerPrivateKeysPath, deployerTokenAccount, ibPortPDA.ToBase58(), burnAmount)
	ValidateError(t, err)
	t.Log("Delegated some tokens to ibport from  deployer")
	t.Log("Creating cross chain transfer tx")

	waitTransactionConfirmations()

	ethReceiverPK, err := ethcrypto.GenerateKey()
	ValidateError(t, err)

	var ethReceiverAddress [32]byte
	copy(ethReceiverAddress[:], ethcrypto.PubkeyToAddress(ethReceiverPK.PublicKey).Bytes())

	t.Logf("#1 EVM Receiver: %v \n", ethcrypto.PubkeyToAddress(ethReceiverPK.PublicKey).String())
	t.Logf("#1 EVM Receiver (bytes): %v \n", ethReceiverAddress[:])

	ibportCreateTransferUnwrapRequestResult, err := ibportExecutor.BuildAndInvoke(
		instructionBuilder.CreateTransferUnwrapRequest(ethReceiverAddress, 2.22274234),
	)
	ValidateError(t, err)
	t.Logf("#1 CreateTransferUnwrapRequest - Tx: %v \n", ibportCreateTransferUnwrapRequestResult.TxSignature)

	ethReceiverPK, err = ethcrypto.GenerateKey()
	ValidateError(t, err)

	copy(ethReceiverAddress[:], ethcrypto.PubkeyToAddress(ethReceiverPK.PublicKey).Bytes())

	t.Logf("#2 EVM Receiver: %v \n", ethcrypto.PubkeyToAddress(ethReceiverPK.PublicKey).String())
	t.Logf("#2 EVM Receiver (bytes): %v \n", ethReceiverAddress[:])

	ibportCreateTransferUnwrapRequestResult, err = ibportExecutor.BuildAndInvoke(
		instructionBuilder.CreateTransferUnwrapRequest(ethReceiverAddress, 3.23441),
	)
	ValidateError(t, err)
	t.Logf("#2 CreateTransferUnwrapRequest -  Tx: %v \n", ibportCreateTransferUnwrapRequestResult.TxSignature)
}


func TestIBPortAttachValue(t *testing.T) {
	var err error
	deployerPrivateKeysPath := "../private-keys/_test_deployer-pk-deployer.json"
	tokenOwnerPath := "../private-keys/_test_only-token-owner.json"
	ibportProgramPath := "../private-keys/_test_only_ibport-owner.json"

	err = CreatePersistedAccount(deployerPrivateKeysPath, true)
	ValidateError(t, err)
	err = CreatePersistedAccount(tokenOwnerPath, true)
	ValidateError(t, err)
	err = CreatePersistedAccount(ibportProgramPath, true)
	ValidateError(t, err)

	deployerAddress, err := ReadAccountAddress(deployerPrivateKeysPath)
	ValidateError(t, err)

	tokenOwnerAddress, err := ReadAccountAddress(tokenOwnerPath)
	ValidateError(t, err)

	WrappedFaucet(t, tokenOwnerPath, tokenOwnerAddress, 10)
	WrappedFaucet(t, deployerPrivateKeysPath, deployerAddress, 10)
	// err = SystemAirdrop(t, deployerPrivateKeysPath, 10)
	// ValidateError(t, err)
	waitTransactionConfirmations()

	tokenDeployResult, err := CreateToken(tokenOwnerPath)
	ValidateError(t, err)

	tokenProgramAddress := tokenDeployResult.Token.ToBase58()

	deployerTokenAccount, err := CreateTokenAccount(deployerPrivateKeysPath, tokenProgramAddress)
	ValidateError(t, err)
	
	ibportAddressPubkey, ibPortPDA, err := CreatePersistentAccountWithPDA(ibportProgramPath, true, [][]byte{[]byte("ibport")})
	if err != nil {
		fmt.Printf("PDA error: %v", err)
		t.FailNow()
	}
	ibportAddress := ibportAddressPubkey.ToBase58()

	fmt.Printf("token  program address: %s\n", tokenProgramAddress)

	t.Logf("tokenProgramAddress: %v", tokenProgramAddress)
	t.Logf("deployerAddress: %v", deployerAddress)
	t.Logf("tokenOwnerAddress: %v", tokenOwnerAddress)
	t.Logf("ibportAddress: %v", ibportAddress)
	t.Logf("ibPortPDA: %v", ibPortPDA.ToBase58())
	t.Logf("deployerTokenAccount: %v", deployerTokenAccount)

	deployerPrivateKey, err := ReadPKFromPath(t, deployerPrivateKeysPath)
	ValidateError(t, err)
	
	waitTransactionConfirmations()

	_, err = DeploySolanaProgram(t, "ibport", ibportProgramPath, deployerPrivateKeysPath, "../binaries/ibport.so")
	ValidateError(t, err)

	endpoint, _ := InferSystemDefinedRPC()

	portDataAccount, err := GenerateNewAccount(deployerPrivateKey, IBPortAllocation, ibportAddress, endpoint)
	ValidateError(t, err)

	ibportExecutor, err := InitGenericExecutor(
		deployerPrivateKey,
		ibportAddress,
		portDataAccount.Account.PublicKey.ToBase58(),
		"",
		endpoint,
		common.PublicKeyFromString(ibportAddress),
	)
	ValidateError(t, err)

	instructionBuilder := executor.NewIBPortInstructionBuilder()

	mockedNebulaAddress := common.PublicKeyFromString(deployerAddress)

	waitTransactionConfirmations()
	ibportInitResult, err := ibportExecutor.BuildAndInvoke(
		instructionBuilder.Init(mockedNebulaAddress, common.TokenProgramID),
	)
	ValidateError(t, err)
	t.Logf("IBPort Init: %v \n", ibportInitResult.TxSignature)

	ibportExecutor.SetAdditionalMeta([]types.AccountMeta{
		{ PubKey: common.TokenProgramID, IsWritable: false, IsSigner: false },
		{ PubKey: common.PublicKeyFromString(tokenProgramAddress), IsWritable: true, IsSigner: false },
		{ PubKey: common.PublicKeyFromString(deployerTokenAccount), IsWritable: true, IsSigner: false },
		{ PubKey: ibPortPDA, IsWritable: false, IsSigner: false },
	})
	
	// allow ibport to mint
	err = AuthorizeToken(t, tokenOwnerPath, tokenProgramAddress, "mint", ibPortPDA.ToBase58())
	ValidateError(t, err)
	t.Log("Authorizing ib port to allow minting")
	t.Log("Call attach value ")

	waitTransactionConfirmations()

	swapId := make([]byte, 16)
    rand.Read(swapId)

	t.Logf("Token Swap Id: %v \n", swapId)

	attachedAmount := float64(227)

	t.Logf("15 - Float As Bytes: %v \n", executor.Float64ToBytes(attachedAmount))

	dataHashForAttach := executor.BuildCrossChainMintByteVector(swapId, common.PublicKeyFromString(deployerTokenAccount), attachedAmount)

	ibportCreateTransferUnwrapRequestResult, err := ibportExecutor.BuildAndInvoke(
		instructionBuilder.AttachValue(dataHashForAttach),
	)
	ValidateError(t, err)

	t.Logf("#1 AttachValue - Tx:  %v \n", ibportCreateTransferUnwrapRequestResult.TxSignature)

	t.Logf("Checking for double spend problem \n")

	swapIdSecond := make([]byte, 16)
    rand.Read(swapIdSecond)

	dataHashForAttachSecond := executor.BuildCrossChainMintByteVector(swapIdSecond, common.PublicKeyFromString(deployerTokenAccount), attachedAmount)

	waitTransactionConfirmations()

	ibportCreateTransferUnwrapRequestResult, err = ibportExecutor.BuildAndInvoke(
		instructionBuilder.AttachValue(dataHashForAttachSecond),
	)
	ValidateError(t, err)

	t.Logf("#2 AttachValue - Tx:  %v \n", ibportCreateTransferUnwrapRequestResult.TxSignature)

	waitTransactionConfirmations()

	swapIdThird := make([]byte, 16)
    rand.Read(swapIdThird)

	dataHashForAttachThird := executor.BuildCrossChainMintByteVector(swapIdThird, common.PublicKeyFromString(deployerTokenAccount), attachedAmount)

	waitTransactionConfirmations()

	ibportCreateTransferUnwrapRequestResult, err = ibportExecutor.BuildAndInvoke(
		instructionBuilder.AttachValue(dataHashForAttachThird),
	)
	ValidateError(t, err)

	t.Logf("#3 AttachValue - Tx:  %v \n", ibportCreateTransferUnwrapRequestResult.TxSignature)

	ibportCreateTransferUnwrapRequestResult, err = ibportExecutor.BuildAndInvoke(
		instructionBuilder.AttachValue(dataHashForAttachThird),
	)

	if err != nil {
		t.Logf("Program must fail with error 0x1 \n")
		t.Logf("If so - double spend has been prevented \n")
	}
}

type OperatingAddress struct {
	PublicKey  common.PublicKey
	PrivateKey string
	PKPath     string
}

func NewOperatingAddress(t *testing.T, path string) (*OperatingAddress, error) {
	var err error
	err = CreatePersistedAccount(path, true)

	if err != nil {
		return nil, err
	}

	pubkey, err := ReadAccountAddress(path)
	if err != nil {
		return nil, err
	}

	privateKey, err := ReadPKFromPath(t, path)
	ValidateError(t, err)

	address := &OperatingAddress {
		PublicKey: common.PublicKeyFromString(pubkey),
		PrivateKey: privateKey,
		PKPath:     path,
	}
	// fmt.Printf("address: %v \n", address)
	
	return address, nil
}

func GenerateConsuls(t *testing.T, consulPathPrefix string, count uint) (*[]OperatingAddress, error) {
	result := make([]OperatingAddress, count)
	
	var i uint

	for i < count {
		path := fmt.Sprintf("%v_%v.json", consulPathPrefix, i)
		// consulPubKeys[i] = 
		// consulPKPaths[i] = 
		address, err := NewOperatingAddress(t, path)
		if err != nil {
			return nil, err
		}
		result[i] = *address

		i++
	}

	return &result, nil
}

func ParallelExecution(callbacks []func()) {
	var wg sync.WaitGroup

	wg.Add(len(callbacks))
	for _, fn := range callbacks {
		go func() {
			fn()
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestNebulaSendValueToIBPortSubscriber (t *testing.T) {
	var err error

	deployer, err := NewOperatingAddress(t, "../private-keys/_test_deployer-pk-deployer.json")
	ValidateError(t, err)

	// tokenOwner, err := NewOperatingAddress(t, "../private-keys/_test_only-token-owner.json")
	// ValidateError(t, err)

	// gravityProgram, err := NewOperatingAddress(t, "../private-keys/_test_only-gravity-program.json")
	// ValidateError(t, err)

	nebulaProgram, err := NewOperatingAddress(t, "../private-keys/_test_only-nebula-program.json")
	ValidateError(t, err)

	ibportProgram, err := NewOperatingAddress(t, "../private-keys/_test_only_ibport-program.json")
	ValidateError(t, err)

	const BFT = 3


	WrappedFaucet(t, deployer.PKPath, deployer.PublicKey.ToBase58(), 10)

	consulsList, err := GenerateConsuls(t, "../private-keys/_test_consul_prefix_", BFT)
	ValidateError(t, err)

	endpoint, _ := InferSystemDefinedRPC()

	nebulaDataAccount, err := GenerateNewAccount(deployer.PrivateKey, NebulaAllocation, nebulaProgram.PublicKey.ToBase58(), endpoint)
	ValidateError(t, err)

	nebulaMultisigAccount, err := GenerateNewAccount(deployer.PrivateKey, MultisigAllocation, nebulaProgram.PublicKey.ToBase58(), endpoint)
	ValidateError(t, err)

	fmt.Printf("deploying in parallel 3 contracts \n")

	// _, err = DeploySolanaProgram(t, "gravity", gravityProgram.PKPath, deployer.PKPath, "../binaries/gravity.so")
	// ValidateError(t, err)

	_, err = DeploySolanaProgram(t, "nebula", nebulaProgram.PKPath, deployer.PKPath, "../binaries/nebula.so")
	ValidateError(t, err)

	_, err = DeploySolanaProgram(t, "ibport", ibportProgram.PKPath, deployer.PKPath, "../binaries/ibport.so")
	ValidateError(t, err)

	
	nebulaBuilder := executor.NebulaInstructionBuilder{}
	nebulaExecutor, err := InitGenericExecutor(
		deployer.PrivateKey,
		nebulaProgram.PublicKey.ToBase58(),
		nebulaDataAccount.Account.PublicKey.ToBase58(),
		nebulaMultisigAccount.Account.PublicKey.ToBase58(),
		endpoint,
		common.PublicKeyFromString(""),
	)
	ValidateError(t, err)

	var oracles []byte
	for _, consul := range *consulsList {
		oracles = append(oracles, consul.PublicKey.Bytes()...)
	}

	nebulaInitResponse, err := nebulaExecutor.BuildAndInvoke(
		nebulaBuilder.Init(BFT, nebula.Bytes, common.PublicKeyFromString(""), oracles),
	)

	fmt.Printf("Nebula Init: %v \n", nebulaInitResponse.TxSignature)

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