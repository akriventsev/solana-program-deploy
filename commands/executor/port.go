package executor

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/portto/solana-go-sdk/common"
)

func NewIBPortInstructionBuilder() *IBPortInstructionBuilder {
	return &IBPortInstructionBuilder{}
}

type IBPortInstructionBuilder struct{}

func (port *IBPortInstructionBuilder) Init(nebula, token common.PublicKey) interface{} {
	return struct {
		Instruction       uint8
		NebulaDataAccount common.PublicKey
		TokenDataAccount  common.PublicKey
	}{
		Instruction:       0,
		NebulaDataAccount: nebula,
		TokenDataAccount:  token,
	}
}

func Float64ToBytes(f float64) []byte {
	return float64ToByte(f)
}

func float64ToByte(f float64) []byte {
	//bits := math.Float64bits(f)
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func (port *IBPortInstructionBuilder) CreateTransferUnwrapRequest(receiver [32]byte, amount float64) interface{} {
	fmt.Printf("CreateTransferUnwrapRequest - amount: %v", amount)
	amountBytes := float64ToByte(amount)

	return struct {
		Instruction uint8
		TokenAmount []byte
		Receiver    []byte
	}{
		Instruction: 1,
		TokenAmount: amountBytes,
		Receiver:    receiver[:],
	}
}

func BuildCrossChainMintByteVector(swapId []byte, receiver common.PublicKey, amount float64) []byte {
	var res []byte

	// action
	res = append(res, 'm')
	// swap id
	res = append(res, swapId[0:16]...)
	// amount
	res = append(res, Float64ToBytes(amount)...)
	// receiver
	res = append(res, receiver[:]...)
	
	fmt.Printf("byte array len: %v \n", len(res))
	fmt.Printf("byte array cap: %v \n", len(res))

	return res
}

func (port *IBPortInstructionBuilder) AttachValue(byte_vector []byte) interface{} {
	fmt.Printf("AttachValue - byte_vector: %v", byte_vector)

	return struct {
		Instruction uint8
		ByteVector []byte
	}{
		Instruction: 2,
		ByteVector: byte_vector,
	}
}

// func (port *IBPortInstructionBuilder) TestMint(receiver common.PublicKey, amount float64) interface{} {
// 	amountBytes := float64ToByte(amount)
// 	fmt.Printf("TestMint - amountBytes: %v", amountBytes)

// 	// binary.LittleEndian.

// 	return struct {
// 		Instruction uint8
// 		Receiver    common.PublicKey
// 		TokenAmount []byte
// 	}{
// 		Instruction: 4,
// 		Receiver:    receiver,
// 		TokenAmount: amountBytes,
// 	}
// }
// func (port *IBPortInstructionBuilder) TestBurn(burner common.PublicKey, amount float64) interface{} {
// 	amountBytes := float64ToByte(amount)
// 	fmt.Printf("TestBurn - amountBytes: %v", amountBytes)

// 	return struct {
// 		Instruction uint8
// 		Burner      common.PublicKey
// 		TokenAmount []byte
// 	}{
// 		Instruction: 5,
// 		Burner:      burner,
// 		TokenAmount: amountBytes,
// 	}
// }