/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package types

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
)

func TestHeaderHash(t *testing.T) {
	// 0xcefefd3ade63a5955bca4562ed840b67f39e74df217f7e5f7241a6e9552cca70
	expectedExtra, err := hexutil.HexToBytes("0x0000000000000000000000000000000000000000000000000000000000000000f89af8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b440b8410000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0")
	if err != nil {
		panic(err)
	}

	expectedHash := common.MustHexToHash("0x9e1f014914abc13135c4102917d378f95bce525a1a843a30d8a3ab661a1d7f86")

	// for istanbul consensus
	header := &BlockHeader{Consensus: IstanbulConsensus, ExtraData: expectedExtra}
	if !reflect.DeepEqual(header.Hash(), expectedHash) {
		t.Errorf("expected: %v, but got: %v", expectedHash.Hex(), header.Hash().Hex())
	}

	// append useless information to extra-data
	unexpectedExtra := append(expectedExtra, []byte{1, 2, 3}...)
	header.ExtraData = unexpectedExtra
	if !reflect.DeepEqual(header.Hash(), crypto.MustHash(header)) {
		t.Errorf("expected: %v, but got: %v", crypto.MustHash(header).Hex(), header.Hash().Hex())
	}
}

func TestExtractToIstanbul(t *testing.T) {
	testCases := []struct {
		vanity         []byte
		istRawData     []byte
		expectedResult *IstanbulExtra
		expectedErr    error
	}{
		{
			// normal case
			bytes.Repeat([]byte{0x00}, IstanbulExtraVanity),
			hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0"),
			&IstanbulExtra{
				Validators: []common.Address{
					common.BytesToAddress(hexutil.MustHexToBytes("0x44add0ec310f115a0e603b2d7db9f067778eaf8a")),
					common.BytesToAddress(hexutil.MustHexToBytes("0x294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212")),
					common.BytesToAddress(hexutil.MustHexToBytes("0x6beaaed781d2d2ab6350f5c4566a2c6eaac407a6")),
					common.BytesToAddress(hexutil.MustHexToBytes("0x8be76812f765c24641ec63dc2852b378aba2b440")),
				},
				Seal:          []byte{},
				CommittedSeal: [][]byte{},
			},
			nil,
		},
		{
			// insufficient vanity
			bytes.Repeat([]byte{0x00}, IstanbulExtraVanity-1),
			nil,
			nil,
			ErrInvalidIstanbulHeaderExtra,
		},
	}
	for _, test := range testCases {
		h := &BlockHeader{ExtraData: append(test.vanity, test.istRawData...)}
		istanbulExtra, err := ExtractIstanbulExtra(h)
		if err != test.expectedErr {
			t.Errorf("expected: %v, but got: %v", test.expectedErr, err)
		}
		if !reflect.DeepEqual(istanbulExtra, test.expectedResult) {
			t.Errorf("expected: %v, but got: %v", test.expectedResult, istanbulExtra)
		}
	}
}
