/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	//"context"
	"context"
	"path/filepath"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func newTestSeeleService() *SeeleService {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: filepath.Join(common.GetTempFolder(), "n1"),
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	log := log.GetLogger("seele")

	seeleService, err := NewSeeleService(ctx, conf, log)
	if err != nil {
		panic(err)
	}

	return seeleService
}

func Test_SeeleService_Protocols(t *testing.T) {
	s := newTestSeeleService()
	defer s.Stop()

	protos := s.Protocols()
	assert.Equal(t, len(protos), 1)
}

func Test_SeeleService_Start(t *testing.T) {
	s := newTestSeeleService()
	defer s.Stop()

	s.Start(nil)
	s.Stop()
	assert.Equal(t, s.seeleProtocol == nil, true)
}

func Test_SeeleService_Stop(t *testing.T) {
	s := newTestSeeleService()
	defer s.Stop()

	s.Stop()
	assert.Equal(t, s.chainDB, nil)
	assert.Equal(t, s.accountStateDB, nil)
	assert.Equal(t, s.seeleProtocol == nil, true)

	// can be called more than once
	s.Stop()
	assert.Equal(t, s.chainDB, nil)
	assert.Equal(t, s.accountStateDB, nil)
	assert.Equal(t, s.seeleProtocol == nil, true)
}

func Test_SeeleService_APIs(t *testing.T) {
	s := newTestSeeleService()
	apis := s.APIs()

	assert.Equal(t, len(apis), 6)
	assert.Equal(t, apis[0].Namespace, "seele")
	assert.Equal(t, apis[1].Namespace, "txpool")
	assert.Equal(t, apis[2].Namespace, "download")
	assert.Equal(t, apis[3].Namespace, "network")
	assert.Equal(t, apis[4].Namespace, "debug")
	assert.Equal(t, apis[5].Namespace, "miner")
}
