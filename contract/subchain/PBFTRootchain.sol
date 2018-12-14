pragma solidity ^0.4.24;

// external modules
import "./ByteUtils.sol";
import "./ECRecovery.sol";
import "./SafeMath.sol";
import "./PriorityQueue.sol";

/// @title A PBFT consensus subchain contract in Seele root chain
/// @notice You can use this contract for a PBFT consensus subchain in Seele.
/// @dev The contract is based on the fact that the operator is trustworthy.
/// @author seeledev@seeletech.net
contract PBFTRootchain {
    using SafeMath for uint256;
    using PriorityQueue for uint256[];

    uint8 constant public SIGNATURE_LENGTH = 65;
    uint8 constant public MIN_LENGTH_OPERATOR = 4;
    uint8 constant public MAX_LENGTH_OPERATOR = 21;

    /** @dev Operator related */
    uint256 public opslen;
    mapping(address => uint256) public operators;
    uint128 public operatorDepositBond = 1234567890;

    /** @dev Child Chain related */
    uint256 public currentChildBlockNum;
    mapping(uint256 => ChildBlock) public childBlocks; 
    struct ChildBlock{
        bytes32 root;
        uint256 timestamp;
    }

    /** @dev User related */
    uint256 public userDepositBond = 1234567890;
    uint256 public userExitBond = 1234567890;
    uint256 public exitNonce = 0;
    uint256 public exitTimeLimit = 1 weeks;
    uint256[] public userExitQueue;
    mapping(uint256 => Exit) public userExits;
    struct Exit{
        address user;
        uint256 deposit;
        uint256 bond;
        uint256 timestamp;
    }

    address private _owner;

    /** events */
    event AddOperator(address indexed operator);
    event DeleteOperator(address indexed operator);
    event UserDeposit(address indexed user, uint256 deposit);
    event StartUserExit(address indexed user, uint256 deposit, uint256 bond, uint256 exitNonce);
    event FinalizeUserExit(address indexed user, uint256 amount, uint256 exitNonce);
    event SubmitBlock(address indexed operator, bytes32 root, uint256 timestamp);
    event ChallengedUserExit(address indexed user, uint256 exitNonce, address indexed operator);

    /** @dev Reverts if called by any account other than the owner. */
    modifier onlyOwner() {
        require(msg.sender == _owner, "You're not the owner of the contract");
         _;
    }

    /** @dev Reverts if called by any account other than the operator. */
    modifier onlyOperator() {
        require(operators[msg.sender] > 0, "You're not the operator of the contract");
        _;
    }

    /**
     * @dev The PBFTRootchain constructor sets the original `owner` of the
     * contract to the sender account and initial the operators.
     * @param ops Is the PBFT consensus operators.
     * @param deposits Is the deposits of PBFT consensus operators.
     */
    constructor(address[] ops, uint256[] deposits) public payable{
        require(ops.length >= MIN_LENGTH_OPERATOR && ops.length <= MAX_LENGTH_OPERATOR, "Invalid operators length");
        require(ops.length == deposits.length, "Invalid deposits length");
        uint256 amount = 0;
        for (uint256 i = 0; i < ops.length && isValidAddOperator(ops[i], deposits[i]); i++){
            require(operators[ops[i]] == 0, "Repeated operator");
            operators[ops[i]] = deposits[i];
            amount = amount.add(deposits[i]);
        }
        require(msg.value >= amount, "You don't give me the enough money");
        _owner = msg.sender;
        opslen = ops.length;
    }

    /** @return the address of the owner. */
    function owner() public view returns(address) {
        return _owner;
    }

    /** @dev Transfers the current balance to the owner and terminates the contract. */
    function destroy() onlyOwner public {
        // todo need release all the debts
        selfdestruct(_owner);
    }

    /**
     * @dev Verify that the transaction has been signed more than  2/3 operators.
     * @param hash Is the hash of the transaction
     * @param ops Is the operators of the transaction
     * @param sigs Is the signatures of the transaction operators
     */
    function isVerify(bytes32 hash, address[] ops, bytes sigs) public view returns(bool){
        require(ops.length * 3 >= opslen * 2, "The operators is less than 2/3");
        require(sigs.length % SIGNATURE_LENGTH == 0 && sigs.length == ops.length * SIGNATURE_LENGTH, "Invalid signatures length");

        for (uint256 i = 0; i < ops.length; i++){
            if (operators[ops[i]] > 0) {
                return false;
            }

            bytes memory sig = ByteUtils.slice(sigs, i * SIGNATURE_LENGTH, SIGNATURE_LENGTH);
            if (ops[i] != ECRecovery.recover(hash, sig)) {
                return false;
            }
        }
        return true;
    }

    /**
     * @dev Verify that the operator is valid and that the deposit is sufficient
     * @param operator Is the operator of the transaction
     * @param deposit Is the deposit of PBFT consensus operator.
     */
    function isValidAddOperator(address operator, uint256 deposit) public view returns(bool){
        require(operator != address(0), "Invalid operator address");
        require(deposit >= operatorDepositBond, "Insufficient operator deposit value");

        return true;
    }

    /**
     * @dev Add a valid operator with more than 2/3 operator signatures.
     * @param operator Is the operator of the transaction
     * @param hash Is the hash of the transaction
     * @param ops Is the operators of the transaction
     * @param sigs Is the signatures of the transaction operators
     */
    function addOperator(address operator, bytes32 hash, address[] ops, bytes sigs) onlyOperator public payable {
        require(isValidAddOperator(operator, msg.value), "Invalid add a operator");
        require(operators[operator] == 0, "Duplicate operator address");
        require(isVerify(hash, ops, sigs), "Invalid verification signatures");
        require(opslen.add(1) <= MAX_LENGTH_OPERATOR, "More than MAX_LENGTH_OPERATOR");

        operators[operator] = msg.value;
        opslen = opslen.add(1);
        emit AddOperator(operator);
    }

    /**
     * @dev Delete a valid operator with more than 2/3 operator signatures.
     * @param operator Is the operator of the transaction
     * @param hash Is the hash of the transaction
     * @param ops Is the operators of the transaction
     * @param sigs Is the signatures of the transaction operators
     */
    function deleteOperator(address operator, bytes32 hash, address[] ops, bytes sigs) onlyOperator public payable {
        require(operators[operator] > 0, "Operator address that does not exist");
        require(isVerify(hash, ops, sigs), "Invalid verification signatures");
        require(opslen.sub(1) >= MIN_LENGTH_OPERATOR, "Less than MIN_LENGTH_OPERATOR");
        require(address(this).balance >= operators[operator], "I don't have enough money to pay this delete operator");

        operator.transfer(operators[operator]);
        delete operators[operator];
        opslen = opslen.sub(1);
        emit DeleteOperator(operator);
    }

    /**
     * @dev User deposits into the root chain to join the child chain
     */
    function deposit() public payable {
        require(msg.value >= userDepositBond, "Insufficient user deposit value");
        emit UserDeposit(msg.sender, msg.value);
    }

    /**
     * @dev The user starts to exit the value of the subchain
     * @param value User exits value
     */
    function startExit(uint256 value) public payable {
        require(msg.value >= userExitBond, "Insufficient user exit value");

        uint256 nonce = exitNonce;
        exitNonce = exitNonce.add(1);

        Exit memory exit = Exit({
            user: msg.sender,
            deposit: value,
            bond: msg.value,
            timestamp: block.timestamp
        });
        userExits[nonce] = exit;
        userExitQueue.insert(nonce);

        emit StartUserExit(exit.user, exit.deposit, exit.bond, nonce);
    }

    /**
     * @notice Finalizing is an expensive operation if the queue is large
     * @dev finalize All valid exits
     */
    function finalizeExits() public onlyOperator{
        require(userExitQueue.currentSize() > 0, "All user exits have been finalized");

        uint256 nonce = userExitQueue.getMin();
        Exit memory exit = userExits[nonce];
        while(block.timestamp.sub(exit.timestamp) >= exitTimeLimit){
            if (exit.deposit > 0){
                uint256 amount = exit.deposit.add(exit.bond);
                require(address(this).balance >= amount, "I don't have enough money to pay this finalize amount");
                exit.user.transfer(amount);
                emit FinalizeUserExit(exit.user, amount, nonce);

                delete userExits[nonce];
            }

            userExitQueue.delMin();

            if (userExitQueue.currentSize() == 0){
                break;
            }

            nonce = userExitQueue.getMin();
            exit = userExits[nonce];
        }
    }

    /**
     * @dev Used to challenge users to exit illegally. If successful, the user
     * exits with a failure and the bond is confiscated to the challenger.
     * @param u The user who exits
     * @param nonce The nonce of user exits
     * @notice The operator must be reliabale
     */
    function challengeUserExit(address u, uint256 nonce) public onlyOperator{
        Exit memory exit = userExits[nonce];
        require(exit.user == u && exit.deposit > 0, "This user exit could not be found or has been finalized");
        require(block.timestamp - exit.timestamp < exitTimeLimit, "This user exit has exceeded the challenge period");

        delete userExits[nonce];

        require(address(this).balance >= userExitBond, "I don't have enough money to pay for this userExitBond challenge");
        msg.sender.transfer(userExitBond);

        emit ChallengedUserExit(exit.user, nonce, msg.sender);
    }

    /**
     * @dev Used to submit child block, you don't have to submit child block number one by one.
     * @param blockNum The child block number
     * @param r The merkle tree root hash of child block
     */
    function submitBlock(uint256 blockNum, bytes32 r) public onlyOperator{
        currentChildBlockNum = blockNum;
        childBlocks[blockNum] = ChildBlock({
            root: r,
            timestamp: block.timestamp
        });

        emit SubmitBlock(msg.sender, r, block.timestamp);
    }

}
