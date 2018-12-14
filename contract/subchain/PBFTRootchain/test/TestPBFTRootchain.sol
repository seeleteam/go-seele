pragma solidity ^0.4.2;

import "truffle/Assert.sol";
import "truffle/DeployedAddresses.sol";
import "../contracts/PBFTRootchain.sol";

contract TestPBFTRootchain {

  function testInitial() public {
    PBFTRootchain meta = PBFTRootchain(DeployedAddresses.PBFTRootchain());

    uint expected = 4;
    Assert.equal(meta.opslen(), expected, "operators length is 4");
	
	address o = 0xCA35b7d915458EF540aDe6068dFe2F44E8fa733c;
	uint256 deposit = 1234567890;
    Assert.equal(meta.operators(o), deposit, "operator[0xCA35b7d915458EF540aDe6068dFe2F44E8fa733c] = 1234567890");
  }
}
