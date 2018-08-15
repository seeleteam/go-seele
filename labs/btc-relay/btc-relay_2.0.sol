/*

Implements BTC-RELAY:

The latest updated:
    time - 08/10/2018
    author - Feifan Wang
    version - 2.0

*/
pragma solidity ^0.4.0;

contract BTCRelay {

    struct Block{
        uint256   blockHeaderBytes;
        int256    height;
        uint256   previousblockhash;
        uint256[] txs;
    }

    uint256                   public PreBlockHash;
    int256                    public PreBlockHeight = -1;
    mapping(uint256 => Block) public Blocks;

    uint256 constant FEE = 1;

    event StoreHeader(uint256 blockHash, int256 height, uint256 previousblockhash, uint256[] txs, uint256 returnCode);
    event GetHeader(uint256 blockHash, uint256 returnCode);
    event VerifyTransaction(uint256 txHash, uint256 returnCode);
    event RelayTransaction(uint256 txHash, uint256 returnCode);
    event Error(string errType, uint256 code, string message);

    address private trustedXRelay;

    constructor() public{
        trustedXRelay = msg.sender;
    }

    // 1. verification of a txBlockHash transaction
    function verifyTx(uint256 txBlockHash, uint256 txHash) payable public returns (uint256) {
        uint256 returnCode = 0;

        if (msg.value < FEE) {
            emit Error("verifyTx", returnCode, "ERR_MONEY_ISNOT_ENOUGH");
            return returnCode;
        }

        if (PreBlockHeight < 6 || Blocks[txBlockHash].height < PreBlockHeight - 6) {
            emit Error("verifyTx", returnCode, "ERR_CONFIRMATIONS_LESS_THAN_6");
            return returnCode;
        }

        // Send money to X relayer
        trustedXRelay.transfer(msg.value);

        uint256[] memory txs = Blocks[txBlockHash].txs;
        for (uint i = 0; i < txs.length; i++) {
            if (txs[i] == txHash){
                returnCode = 1;
                break;
            }
        }
        emit VerifyTransaction(txHash, returnCode);

        return returnCode;
    }

    // 2. optionally relay the Bitcoin transaction to any Seele contract
    function relayTx(uint256 txBlockHash, uint256 txHash, address contractAddr) payable public returns(uint256){
        uint256 returnCode = 0;

        uint256 ok = verifyTx(txBlockHash, txHash);
        if (ok != 0) {
            BitcoinProcessor processor = BitcoinProcessor(contractAddr);
            processor.processTransaction(txBlockHash, txHash);

            returnCode = 1;
            emit RelayTransaction(txHash, returnCode);
            return returnCode;
        }

        emit Error("RelayTransaction", returnCode, "ERR_RELAY_VERIFY");
        return returnCode;
    }

    // 3. storage of Bitcoin block headers
    function storeBlockHeader(uint256 blockHash, int256 height, uint256 previousblockhash, uint256[] txs) public returns(uint256){
        uint256 returnCode = 0;

        if (msg.sender != trustedXRelay){
            emit Error("storeBlockHeader", returnCode, "ERR_STORE_X_RELAYER");
            return returnCode;
        }

        if (previousblockhash != PreBlockHash) {
            emit Error("storeBlockHeader", returnCode, "ERR_NO_PREV_BLOCK");
            return returnCode;
        }

        if (height <= PreBlockHeight){
            emit Error("storeBlockHeader", returnCode, "ERR_BLOCK_ALREADY_EXISTS");
            return returnCode;
        }

        if (Blocks[blockHash].height > 0){
            emit Error("storeBlockHeader", returnCode, "ERR_BLOCK_HASH_ALREADY_EXISTS");
            return returnCode;
        }

        if (txs.length == 0){
            emit Error("storeBlockHeader", returnCode, "ERR_TRANSACTIONS_IS_EMPTY");
            return returnCode;
        }

        Blocks[blockHash] = Block(blockHash, height, previousblockhash, txs);
        PreBlockHash = blockHash;
        PreBlockHeight = height;
        returnCode = 1;
        emit StoreHeader(blockHash, height, previousblockhash, txs, returnCode);

        return returnCode;
    }

    // 4. inspection of the latest Bitcoin block header stored in the contract
    function getBlockHeader(uint256 blockHash) public payable returns(uint256) {
        uint256 returnCode = 0;
        if (msg.value < FEE) {
            emit Error("getBlockHeader", returnCode, "ERR_MONEY_ISNOT_ENOUGH");
            return returnCode;
        }

        // Send money to X relayer
        trustedXRelay.transfer(msg.value);

        if (Blocks[blockHash].height > 0){
            returnCode = 1;
        }
        emit GetHeader(blockHash, returnCode);

        return returnCode;
    }
}


contract BitcoinProcessor {
    uint256 public lastTxHash;
    uint256 public ethBlock;

    address private _trustedXRelay;

    constructor(address trustedXRelay) public{
        _trustedXRelay = trustedXRelay;
    }

    // processTransaction should avoid returning the same
    // value as ERR_RELAY_VERIFY to avoid confusing callers
    //
    // this exact function signature is required as it has to match
    // the signature specified in BTCRelay (otherwise BTCRelay will not call it)
    function processTransaction(uint256 blockHash, uint256 txHash) payable public returns (int256) {
        // log0("processTransaction called");

        // only allow trustedXRelay, otherwise anyone can provide a fake txn
        if (msg.sender == _trustedXRelay) {
            // log1("processTransaction blockHash, ", bytes32(blockHash));
            // log1("processTransaction txHash, ", bytes32(txHash));
            ethBlock = block.number;
            lastTxHash = txHash;
            // parse & do whatever with txn
            // For example, you should probably check if txHash has already
            // been processed, to prevent replay attacks.
            return 1;
        }

        // log0("processTransaction failed");
        return 0;
    }
}
