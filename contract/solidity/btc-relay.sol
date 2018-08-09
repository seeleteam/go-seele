/*

Implements BTC-RELAY: https://github.com/ethereum/btcrelay

Test data:
    verifyTx - "564aec36d5eefe52021f4b8b6419c9dca42f1d03b4c7a6034dfbbd1537f2a2ea",[],[],12
    relayTx - "564aec36d5eefe52021f4b8b6419c9dca42f1d03b4c7a6034dfbbd1537f2a2ea",[],[],12, "0xff0fb1e59e92e94fac74febec98cfd58b956fa61"
    storeBlockHeader - "671bf9e29adb3a2618fac9e51251f03f4b4fe664aeb0bc404b7b9c69f8401c4c"
    getBlockHeader - 671bf9e29adb3a2618fac9e51251f03f4b4fe664aeb0bc404b7b9c69f8401c4c

TODO strings:
    1. sha256
    2. block verify
    3. call other contract(maybe it is not ok, cause the verifyTx is enough?)
    4. compute merkle root

The latest updated: 
    time - 08/09/2018
    author - Feifan Wang

*/
pragma solidity ^0.4.0;

contract BTCRelay {
    uint256 preBlockHash;
    mapping(uint256 => bytes) public blocks;
    uint256 FEE = 1;

    event StoreHeader(uint256 blockHash, uint256 returnCode);
    event GetHeader(bytes blockHeaderBytes, uint256 returnCode);
    event VerifyTransaction(uint256 txHash, uint256 returnCode);
    event RelayTransaction(uint256 txHash, uint256 returnCode);
    event Error(string errType, uint256 code, string message);

    // 1. verification of a Bitcoin transaction
    function verifyTx(bytes txBytes, uint256[] txIndex, uint256[] sibling, uint256 txBlockHash) payable public returns(uint256){
        uint256 returnCode = 0;
        if (msg.value < FEE) {
            emit Error("verifyTx", 0, "ERR_MONEY_ISNOT_ENOUGH");
            return 0;
        }

        if (txBytes.length != 64) {
            emit Error("VerifyTransaction", 0, "ERR_TX_64BYTE");
            return 0;
        }
        uint256 txHash = sha256(txBytes);

        returnCode = verify(txHash, txIndex, sibling, txBlockHash, msg.value);
        emit VerifyTransaction(txHash, returnCode);

        if (returnCode == 1) {
            return txHash;
        }

        return returnCode;
    }

    // 2. optionally relay the Bitcoin transaction to any Ethereum contract
    function relayTx(bytes txBytes, uint256[] txIndex, uint256[] sibling, uint256 txBlockHash, address contractAddr) payable public returns(uint256){
        uint256 returnCode = 0;
        uint256 txHash = verifyTx(txBytes, txIndex, sibling, txBlockHash);
        if (txHash != 0) {
            // TODO call contractAddr with txBytes and txHash
            returnCode = 1;
            emit RelayTransaction(txHash, returnCode);
            return(returnCode);
        }

        emit Error("RelayTransaction", returnCode, "ERR_RELAY_VERIFY");
        return returnCode;
    }

    // 3. storage of Bitcoin block headers
    function storeBlockHeader(bytes blockHeaderBytes) public returns(uint256){
        uint256 returnCode = 0;
        uint256 blockHash = sha256(blockHeaderBytes);
        if (blocks[blockHash].length != 0){
            emit Error("storeBlockHeader", 0, "ERR_BLOCK_ALREADY_EXISTS");
            return returnCode;
        }

        // TODO Check the blockHash by Difficulty and preBlockHash
        if (true) {
            returnCode = 1;
            blocks[blockHash] = blockHeaderBytes;
            preBlockHash = blockHash;
            emit StoreHeader(blockHash, 1);
        }

        return returnCode;
    }

    // 4. inspection of the latest Bitcoin block header stored in the contract
    function getBlockHeader(uint256 blockHash) public payable returns(bytes){
        uint256 returnCode = 0;
        if (msg.value < FEE) {
            emit Error("getBlockHeader", 0, "ERR_MONEY_ISNOT_ENOUGH");
            return "0";
        }
        returnCode = 1;

        bytes memory blockHeaderBytes = blocks[blockHash];
        emit GetHeader(blockHeaderBytes, returnCode);
        return blockHeaderBytes;
    }

    function verify(uint256 txHash, uint256[] txIndex, uint256[] sibling, uint256 txBlockHash, uint256 value) private returns(uint256){
        /*  TODO
            Check if tx has 6 confirmations
            Check if the blockHash is in the main chain

            merkle = computeMerkle(txHash, txIndex, sibling)
            realMerkleRoot = getMerkleRoot(txBlockHash)
            bool = merkle == realMerkleRoot
        */
        if (true) {
            return 1;
        }

        return 0;
    }

    function sha256(bytes bs) private returns(uint256){
        // TODO
        // flip32Bytes(sha256(sha256($dataBytes:str)))
        emit Error("sha256", 1, "ERR_SHA256_NOTCOMPELED");
        return 0x4b974d903d5c112d13546dc34e48f2c84938b0bcef67425160961828bc36cf4d;
    }
}
