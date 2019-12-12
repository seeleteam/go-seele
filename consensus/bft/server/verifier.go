package server

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

func ContainVer(addres []common.Address, ver common.Address) (int, bool) {
	for i, addr := range addres {
		if ver == addr {
			return i, true
		}
	}
	return -1, false
}

func (s *server) GetVerifierFromSWExtra(header *types.BlockHeader, addrs []common.Address, auths []bool) error {

	// get the new verifiers from n-1th block secondwitness
	// check the verifier from previous deposit or exit txs:
	var swExtra *types.SecondWitnessExtra
	// err := swExtra.ExtractSWExtra(header)
	swExtra, err := types.ExtractSecondWitnessExtra(header)
	if err != nil {
		s.log.Error("failed to extract secondwitness extra data, err : %s", err)
		return err
	}
	// in case one same verifier deposit and exit at the same block,
	// make usre add before delect
	for _, depVer := range swExtra.DepositVers {
		if _, contain := ContainVer(addrs, depVer); contain {
			s.log.Warn("verifier from deposit tx already exit in verifier set")
			continue
		}
		addrs = append(addrs, depVer)
		auths = append(auths, true)
	}
	for _, exitVer := range swExtra.ExitVers {
		if _, exist := ContainVer(addrs, exitVer); exist {
			// addrs = append(addrs[:i], addrs[i+1:]...)
			// auths = append(auths[:i], auths[i+1:]...)
			s.log.Warn("verifier from deposit tx already exit in verifier set")
			continue
		}
		addrs = append(addrs, exitVer)
		auths = append(auths, false)
	}
	return nil
}

func (s *server) GetCurrentVerifiers(verset []common.Address, addrs []common.Address, auths []bool) []common.Address {
	for i, addr := range addrs {
		if auths[i] {
			if _, contain := ContainVer(verset, addr); !contain {
				verset = append(verset, addr)
			}
		}
	}
	return verset
}
