pragma solidity ^0.4.24;

// import external modules here
import './Challenge.sol';
import './Transaction.sol';
import './PriorityQueue.sol';
import './SafeMath.sol';

/** The template of one contract. Please foloow the order to create one contract 
 * refer to: solidity offical website https://solidity.readthedocs.io/en/v0.4.24/style-guide.html?highlight=private%20view#
contract A {
    function A() public {
        ...
    }

    function() public {
        ...
    }

    // External functions
    // ...

    // External functions that are view
    // ...

    // External functions that are pure
    // ...

    // Public functions
    // ...

    // Internal functions
    // ...

    // Private functions
    // ...
}
*/


/**
@title ZKRootChain: zero knowledge proot subchain smart contract
@notice: deposit, challege, exit
@dev 
*/
contract ZKRootChain {
   
    using Challenge for Challenge.chanllenge[];
    using Transactions for bytes;
    using PriorityQueue for uint256[];
    using SafeMath for uint256;

    /**@dev constant value */
    uint8 constant public SIGNATURE_LEN = 65;
    uint256 constant public SAFEDEPOSITSCALE = uint(2); // safedepositscale = (apply + deposit) / apply
    //uint256 constant public IPOMax
    
    /**@dev owner */
    address private _owner;

    /**@dev operator */
    uint256 public opLen;
    uint256 public miniDeposit;
    
    mapping (address => uint256) public operators;

    /**@dev user */
    uint256 public userDepositBond = 0;
    mapping (uint256 => state) public states
    uint256 public depositCount;

    struct state {
        address user;
        bool hasBalance; // refill or new user;
        bool isConfirmed;
        uint256 balance;
        uint256 nonce;
        uint256 lastSentHeight;
    }

    /** Exit Process */
    uint256 public userExitBond = 0;
    uint256 public userExitNonce = 0;
    uint256 public userExitTimeLimit = 2 days;
    uint256[] public userExitQueue;
    mapping(uint256 => exit) public exits;
    struct exit {
        address user;
        address owner;
        uint256 balance;
        uint256 nonce;
        uint256 lastSentHeight;
        uint256 lastUpdatedHeight;
        bytes merkleProof;
        uint256 timestamp;
    }

    /**subchain block structure */
    uint256 currentblkNum;
    mapping(uint256 => subchainBlock) public suchainBlocks;
    struct subchainBlock {
        bytes32 prevRootHash; //old state
        bytes32 rootHash; // new state
        uint256 timestamp;
        bool hasDepositTx; // once there is any deposit tx, need to audit to mainnet immediately;
        uint256 totalFee;
    }
    /**@dev events will be triggered automatically once meets the conditions*/
    event UserDeposit (address depositor, uint256 amount, uint356 uid);
    event StartUserExit(address user, uint256 deposit, uint256 bond, uint256 exitNonce);
    event FinalizedUserExit(address user, uint256 amount, uint256 exitNonce);
    event ChallengeUserExit(address user, uint256 exitNonce, address operator);
    event Audit(address user, address operator);


    /**@dev Restrict some access to only operator or owner*/
    modifier onlyOwer() {
        require (
            msg.sender == _owner,
            "Only owner can call this"
        );
        _;
    }
    modifier onlyOperator() {
        require(
            operator[msg.sender] > 0,
            "only operator can call this"
        );
        _;
    }

    /**
     * @dev Constructor will construct the original parameter for subchain: ownership, deposit, consensus, etc
     * ZKSubchain each user is the operator who can packet the transaction to owner to submit a block
     * all operators with related desposit will be used to intiate the subchain
     * operators map all operators of subchain with their deposit
     */
    constructor(address[] ops, uint256[] deposits) public {
        uint256 amount = 0; // the total amount of subchain deposit
        for (uint256 i = 0; i < ops.length && isValidOperator(ops[i], deposits[i]); i++) {
            operators[ops[i]] = deposits[i]; 
            amount = amount.add(deposits[i])
        }
        require(msg.value >= amount.mul(SAFEDEPOSITSCALE)); 
        _owner = msg.sender;
        opslen = ops.length;
    }

    

    function owner() public view returns(address) {
        return _owner;
    }
    function destroy() onlyOwner public {
        // check whether all bonds are refund to operator. yes: safe destroy; no: refund to operator first and then safe refund;
    }

    /**
     @dev deposit: normal user deposits into subchain
     @param owner The owner of subchain
     @param amount Deposit amount
     @return   Return deposit uid
     */
    function deposit(address owner, uint256 amount, uint256 height) public payable returns (uint256){
        // require(isValidOperator(operator, amount), "Invalid operator")
        uint256 uid = uint256(keccak256(owner, msg.sender, operators.length));
        sates[uid] = state({
            user: operator; // in ZKSubchain, there are two types of game players: owner: own subchain, operatore: normal user;
            hasBalance: true;
            isConfirmed: false;
            balance: amount;
            nonce: 0; // TODO
            lastSentHeight: height; //TODO
        })
        depositCount += 1;
        emit UserDeposit(msg.sender, amount, uid);
        return uid;
    }

    /**
    @dev abortDeposit: exit a deposit if anything unusual happens
    @param uid deposit uid
    */
    function abortDeposit(uint256 uid) {
        require(!states[uid].isConfirmed, "Can not abort the confirmed deposit"); // the deposit must be not confirmed
        require(states[uid] == msg.sender, "Wrong deposit uid to abort"); // sender must be the depositor
        // abort the deposit
        msg.sender.transfer(states[uid].balance);
        delete states[uid].hasBalance;
    }

    /** 
    @dev exit will consume some gas
    */
    function startExit(address[] exts) public payable {
        // require some gas to exit
        require(msg.value > userExitBond, "Insufficient exit value");
        // TODO do we need check the format of exit?
        uint256 eid = uint256()
    }

    function challengeExit() public {

    }
    function challengeAudit() public {

    }
    function handleChallengeExit() public {

    }
    function 

    function fianalizeEixt () public {}

    // isValidOperatore: check the operator's validity
    function isValidOperator(address operator, uint256 deposit) public view returns(bool) {
        require(operator != address(0), "Invalid operator address!");
        require(operator[ops[i]] == 0, "Operator already exit!");
        require(deposit >= miniDeposit, "Insufficient deposit");
        return true;
    }


    /**
     struct subchainBlock {
        uint256 blkNum;
        bytes32 prevRootHash; //old state
        bytes32 rootHash; // new state
        uint256 timestamp;
        bool hasDepositTx; // once there is any deposit tx, need to audit to mainnet immediately;
        uint256 totalFee;
    }
     */
    function submitBlock(uint256 blkNum, bytes32 preRHash, bytes32 rHash, bool hasDepTx, uint256 tFee) public onlyOwner {
        currentblkNum = blkNum;
        require (depositNumber + currentUserAccount <= 2**depth - 1) // TODO
        subchainBlocks[blkNum] = subchainBlock({
            preRootHash: preRHash,
            rootHash: rHash,
            timeStamp: block.timestamp,
            hasDepositTx: hasDepTx,
            totalFee: tFee
        });
        if (hasDepTx) { // need to audit
            //TODO audit roothash to mainnet.
        }
        emit SubmitBlock(msg.sender, rHash, block.timestamp)
    }
    function isChallended (uint256 uid, bytes challengeTx) public returns (bool) {
        return challenges[uid].contains(challengeTx);
    }
}