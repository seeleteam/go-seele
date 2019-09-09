pragma solidity ^0.4.24;

/**
    @title challenge module
    @author seele team  
 */

 library Challenge {
     struct challenge {
         bool hasValue; // challenge needs to consume gas;
         bytes challengeTx; // challenge will be a tx
         uint256 challengeTxBlkNum;
     }
     function contains()
 }