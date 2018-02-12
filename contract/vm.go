package contract

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

// VM is the interface of smart contract virtual machine.
type VM interface {
	Execute(code []byte, msg []byte)
}

type exeVM struct {
}

func (vm *exeVM) Execute(code []byte, msg []byte) {
	fmt.Println("Smart contract execution begin ...")

	exeFile := tempImage(code, "SmartContract-", ".exe")
	fmt.Println("temp exe file", exeFile)

	cmd := exec.Command(exeFile)
	output, err := cmd.Output()

	// TODO broadcast the execution result (ret/err) to P2P network for consensus
	if err != nil {
		fmt.Println("Failed to execute smart contract. ", err.Error())
	} else {
		fmt.Println("Smart contract execution finished.")
		fmt.Println(string(output))
	}
}

func tempImage(code []byte, prefix, extension string) string {
	fileName := prefix + uuid.New().String() + extension
	filePath := filepath.Join(os.TempDir(), fileName)
	ioutil.WriteFile(filePath, code, 0666)
	return filePath
}
