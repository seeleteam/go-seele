package node

import 	(
	"testing"
	"github.com/magiconair/properties/assert"

)

func Test_LoadConfigFromFilea(t *testing.T) {

	configFilePath := "D:/go/src/github.com/seeleteam/go-seele/cmd/node/config/node1.json"
	genesisConfigFilePath := "D:/go/src/github.com/seeleteam/go-seele/cmd/node/config/genesis.json"
	config,err := LoadConfigFromFile(configFilePath, genesisConfigFilePath)
	if(err == nil){
		assert.Equal(t, config.HTTPServer.HTTPAddr, "127.0.0.1:65027")
	}
}
