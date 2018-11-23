pragma solidity ^0.4.25;

// external modules
import "./ByteUtils.sol";
import "./ECRecovery.sol";
import "./SafeMath.sol";

/// @title A PBFT consensus subchain contract in Seele root chain
/// @notice You can use this contract for a PBFT consensus subchain in Seele.
/// @dev The contract is based on the fact that the operator is trustworthy.
/// @author Wang Feifan (314130948@qq.com)
contract PBFTRootchain {
    using SafeMath for uint256;

    uint8 constant public SIGNATURE_LENGTH = 65;
    uint8 constant public MIN_LENGTH_OPERATOR = 1;
    uint8 constant public MAX_LENGTH_OPERATOR = 21;

    /** @dev Operator related */ 
    uint256 public opslen;
    mapping(address => uint256) public operators;
    uint128 public minOperatorBond = 1234567890;
    
    address private _owner;
    
    /** events */
    event OperatorAdded(address indexed operator);
    event OperatorDeleted(address indexed operator);
    
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
    constructor(address[] ops, uint256[] deposits) public {
        require(ops.length >= MIN_LENGTH_OPERATOR && ops.length <= MAX_LENGTH_OPERATOR, "Invalid operators length");
        require(ops.length == deposits.length, "Invalid deposits length");
        for (uint256 i = 0; i < ops.length && isValidAddOperator(ops[i], deposits[i]); i++){
            operators[ops[i]] = deposits[i];
        }
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
        require(deposit >= minOperatorBond, "Insufficient deposit value");
        
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
        emit OperatorAdded(operator);
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

        operators[operator] = 0;
        opslen = opslen.sub(1);
        emit OperatorDeleted(operator);
    }
}
