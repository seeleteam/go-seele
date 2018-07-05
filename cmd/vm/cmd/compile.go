/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
)

type solCompileOutput struct {
	HexByteCodes    string
	FunctionHashMap map[string]solMethod
}

func (output *solCompileOutput) EncodeRLP(w io.Writer) error {
	value := make([][]byte, 0)

	for k, v := range output.FunctionHashMap {
		value = append(value, []byte(k), common.SerializePanic(v))
	}

	return rlp.Encode(w, value)
}

func (output *solCompileOutput) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}

	var kvs [][]byte
	if err := common.Deserialize(raw, &kvs); err != nil {
		return err
	}

	output.FunctionHashMap = make(map[string]solMethod)

	for i, len := 0, len(kvs)/2; i < len; i++ {
		key := string(kvs[2*i])
		value := kvs[2*i+1]

		m := solMethod{}
		if err := common.Deserialize(value, &m); err != nil {
			return err
		}

		output.FunctionHashMap[key] = m
	}

	return nil
}

// compile compiles the specified solidity file and returns the compilation outputs
// and dispose method to clear the compilation resources. Returns nil if any error occurrred.
func compile(solFilePath string) (*solCompileOutput, func()) {
	if !common.FileOrFolderExists(solFilePath) {
		fmt.Println("The specified solidity file does not exist,", solFile)
		return nil, nil
	}

	// output to temp dir
	tempDir, err := ioutil.TempDir("", "SolCompile-")
	if err != nil {
		fmt.Println("Failed to create temp folder for solidity compilation,", err.Error())
		return nil, nil
	}

	deleteTempDir := true
	defer func() {
		if deleteTempDir {
			os.RemoveAll(tempDir)
		}
	}()

	// run solidity compilation command
	cmdArgs := fmt.Sprintf("--optimize --bin --hashes -o %v %v", tempDir, solFile)
	cmd := exec.Command("solc.exe", strings.Split(cmdArgs, " ")...)
	if err = cmd.Run(); err != nil {
		fmt.Println("Failed to compile the solidity file,", err.Error())
		return nil, nil
	}

	// walk through the temp dir to construct compilation outputs
	output := new(solCompileOutput)
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		switch filepath.Ext(path) {
		case ".signatures":
			output.parseFuncHash(string(content))
		case ".bin":
			output.HexByteCodes = ensurePrefix(string(content), "0x")
		}

		return nil
	}

	if err = filepath.Walk(tempDir, walkFunc); err != nil {
		fmt.Println("Failed to walk through the compilation temp folder,", err.Error())
		return nil, nil
	}

	deleteTempDir = false

	return output, func() {
		os.RemoveAll(tempDir)
	}
}

func (output *solCompileOutput) parseFuncHash(content string) {
	output.FunctionHashMap = make(map[string]solMethod)

	for _, line := range strings.Split(content, "\n") {
		// line: funcHash: method(type1,type2,...)
		if line = strings.Trim(line, "\r"); len(line) == 0 {
			continue
		}

		// add mapping: funcFullName <-> hash
		funcHash := line[:8]
		funcFullName := string(line[10:])
		method := newSolMethod(funcFullName, funcHash)

		output.FunctionHashMap[funcFullName] = method
	}
}

func (output *solCompileOutput) getMethodByName(name string) *solMethod {
	if v, ok := output.FunctionHashMap[name]; ok {
		return &v
	}

	var method solMethod

	for _, v := range output.FunctionHashMap {
		if v.ShortName != name {
			continue
		}

		if len(method.ShortName) > 0 {
			fmt.Println("Short method name not supported for overloaded methods:")
			output.funcHashesUsage()
			return nil
		}

		method = v
	}

	if len(method.ShortName) == 0 {
		fmt.Println("Cannot find the specified method name, please call below methods:")
		output.funcHashesUsage()
	}

	return &method
}

func (output *solCompileOutput) funcHashesUsage() {
	for k := range output.FunctionHashMap {
		fmt.Printf("\t%v\n", k)
	}
}
