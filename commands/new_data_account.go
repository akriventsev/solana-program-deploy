package commands

import (
	"crypto/ed25519"
	"fmt"
	"log"

	"github.com/mr-tron/base58"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/sysprog"
	"github.com/portto/solana-go-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	newDataAccPrivateKey string
	programPrivateKey    string

	space uint64
	// alias for show
	newDataAccCmd = &cobra.Command{
		Hidden: false,

		Use:   "attach",
		Short: "Display a file from the hoarder storage",
		Long:  ``,
		Run:   newAcc,
	}
)

// init
func init() {
	newDataAccCmd.Flags().StringVarP(&programID, "program", "p", "", "Program ID")
	viper.BindPFlag("program", SolanoidCmd.Flags().Lookup("program"))
	newDataAccCmd.MarkFlagRequired("program")

	newDataAccCmd.Flags().StringVarP(&newDataAccPrivateKey, "private-key", "k", "", "private key in base58 encoding")
	viper.BindPFlag("private-key", SolanoidCmd.Flags().Lookup("private-key"))
	newDataAccCmd.MarkFlagRequired("private-key")

	newDataAccCmd.Flags().Uint64VarP(&space, "space", "s", 4, "space for data")
	viper.BindPFlag("space", SolanoidCmd.Flags().Lookup("space"))
	newDataAccCmd.MarkFlagRequired("space")

	SolanoidCmd.AddCommand(newDataAccCmd)
}

func newAcc(ccmd *cobra.Command, args []string) {
	pk, err := base58.Decode(newDataAccPrivateKey)
	if err != nil {
		zap.L().Fatal(err.Error())
	}
	account := types.AccountFromPrivateKeyBytes(pk)

	// pk, err = base58.Decode(programPrivateKey)
	// if err != nil {
	// 	zap.L().Fatal(err.Error())
	// }

	program := common.PublicKeyFromString(programID)

	c := client.NewClient(client.TestnetRPCEndpoint)

	res, err := c.GetRecentBlockhash()
	if err != nil {
		log.Fatalf("get recent block hash error, err: %v\n", err)
	}

	newAcc := types.NewAccount()

	rentBalance, err := c.GetMinimumBalanceForRentExemption(space)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	message := types.NewMessage(
		account.PublicKey,
		[]types.Instruction{
			sysprog.CreateAccount(
				account.PublicKey,
				newAcc.PublicKey,
				program,
				rentBalance,
				space,
			),
		},
		res.Blockhash,
	)

	serializedMessage, err := message.Serialize()
	if err != nil {
		log.Fatalf("serialize message error, err: %v\n", err)
	}

	tx, err := types.CreateTransaction(message, map[common.PublicKey]types.Signature{
		account.PublicKey: ed25519.Sign(account.PrivateKey, serializedMessage),
		newAcc.PublicKey:  ed25519.Sign(newAcc.PrivateKey, serializedMessage),
	})
	if err != nil {
		log.Fatalf("generate tx error, err: %v\n", err)
	}

	rawTx, err := tx.Serialize()
	if err != nil {
		log.Fatalf("serialize tx error, err: %v\n", err)
	}

	txSig, err := c.SendRawTransaction(rawTx)
	if err != nil {
		log.Fatalf("send tx error, err: %v\n", err)
	}
	log.Print("Waiting")
	waitTx(txSig)
	log.Print("End waiting")

	log.Println("txHash:", txSig)
	fmt.Printf("Data Acc privake key: %s\n", base58.Encode(newAcc.PrivateKey))
	fmt.Printf("Data account address: %s\n", newAcc.PublicKey.ToBase58())

}
