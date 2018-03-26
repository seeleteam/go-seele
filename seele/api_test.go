/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

func getTmpConfig() *Config {
	acctAddr := crypto.MustGenerateRandomAddress()

	return &Config{
		TxConf:    *core.DefaultTxPoolConfig(),
		NetworkID: 1,
		Coinbase:  *acctAddr,
	}
}

func Test_PublicSeeleAPI(t *testing.T) {
	conf := getTmpConfig()
	ctx := context.WithValue(context.Background(), "DataDir", "./seeleRoot")
	dataDir := ctx.Value("DataDir").(string)
	defer os.RemoveAll(dataDir)
	log := log.GetLogger("seele", true)
	ss, err := NewSeeleService(ctx, conf, log)
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
