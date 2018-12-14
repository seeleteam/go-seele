var ECRecovery = artifacts.require("./ECRecovery.sol");
var PriorityQueue = artifacts.require("./PriorityQueue.sol");
var PBFTRootchain = artifacts.require("./PBFTRootchain.sol");

module.exports = function(deployer) {
  deployer.deploy(ECRecovery);
  deployer.deploy(PriorityQueue);
  deployer.link(ECRecovery, PBFTRootchain);
  deployer.link(PriorityQueue, PBFTRootchain);
  
  // 1. Operator not enough error
  // deployer.deploy(PBFTRootchain, ["0xca35b7d915458ef540ade6068dfe2f44e8fa733c"], ["1234567890"], {value: 8234567890});
  
  // 2. Repeated operator error
  // deployer.deploy(PBFTRootchain, ["0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0xca35b7d915458ef540ade6068dfe2f44e8fa733c"], ["1234567890", "1234567890", "1234567890", "1234567890"], {value: 8234567890});
  
  // 3. Value not enough error
  // deployer.deploy(PBFTRootchain, ["0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0x14723a09acff6d2a60dcdf7aa4aff308fddc160c", "0x4b0897b0513fdc7c541b6d9d7e929c4e5364d2db", "0x583031d1113ad414f02576bd6afabfb302140225"], ["1234567890", "1234567890", "1234567890", "1234567890"], {value: 823456789});

  // 4. Length is not the same error
  // deployer.deploy(PBFTRootchain, ["0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0x14723a09acff6d2a60dcdf7aa4aff308fddc160c", "0x4b0897b0513fdc7c541b6d9d7e929c4e5364d2db", "0x583031d1113ad414f02576bd6afabfb302140225"], ["1234567890", "1234567890", "1234567890", "1234567890", "1234567890"], {value: 8234567890});

  // 5. Insufficient operator deposit value
  // deployer.deploy(PBFTRootchain, ["0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0x14723a09acff6d2a60dcdf7aa4aff308fddc160c", "0x4b0897b0513fdc7c541b6d9d7e929c4e5364d2db", "0x583031d1113ad414f02576bd6afabfb302140225"], ["123456789", "1234567890", "1234567890", "1234567890"], {value: 8234567890});

  
  deployer.deploy(PBFTRootchain, ["0xca35b7d915458ef540ade6068dfe2f44e8fa733c", "0x14723a09acff6d2a60dcdf7aa4aff308fddc160c", "0x4b0897b0513fdc7c541b6d9d7e929c4e5364d2db", "0x583031d1113ad414f02576bd6afabfb302140225"], ["1234567890", "1234567890", "1234567890", "1234567890"], {value: 8234567890});
};
