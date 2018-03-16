/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/log"
)

func getTmpConfig() *Config {
	rootDir, err := ioutil.TempDir("", "seeleRoot")
	if err != nil {
		panic(err)
	}
	accAddr, _ := common.GenerateRandomAddress()

	return &Config{
		txConf:    *core.DefaultTxPoolConfig(),
		NetworkID: 1,
		DataRoot:  rootDir,
		Coinbase:  *accAddr,
	}
}

func Test_PublicSeeleAPI(t *testing.T) {
	conf := getTmpConfig()
	defer os.RemoveAll(conf.DataRoot)
	log := log.GetLogger("seele", true)
	ss, err := NewSeeleService(conf, log)
	if err != nil {
		t.Fatal()
	}

	api := NewPublicSeeleAPI(ss)
	var addr common.Address
	api.Coinbase(nil, &addr)

	if !bytes.Equal(conf.Coinbase[0:], addr[0:]) {
		t.Fail()
	}
}
