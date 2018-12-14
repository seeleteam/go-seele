pragma solidity ^0.4.24;

// external modules
import "./SafeMath.sol";

library PriorityQueue {
    using SafeMath for uint256;

    function insert(uint256[] storage heapList, uint256 k)
        public
    {
        heapList.push(k);
        if (heapList.length > 1)
            percUp(heapList, heapList.length.sub(1));
    }

    function getMin(uint256[] storage heapList)
        public
        view
        returns (uint256)
    {
        require(heapList.length > 0, "empty queue");

        return heapList[0];
    }

    function delMin(uint256[] storage heapList)
        public
        returns (uint256)
    {
        require(heapList.length > 0, "empty queue");

        uint256 min = heapList[0];

        // move the last element to the front
        heapList[0] = heapList[heapList.length.sub(1)];
        delete heapList[heapList.length.sub(1)];
        heapList.length = heapList.length.sub(1);

        if (heapList.length > 1) {
            percDown(heapList, 0);
        }

        return min;
    }

    function minChild(uint256[] storage heapList, uint256 i)
        private
        view
        returns (uint256)
    {
        uint lChild = i.mul(2).add(1);
        uint rChild = i.mul(2).add(2);

        if (rChild > heapList.length.sub(1) || heapList[lChild] < heapList[rChild])
            return lChild;
        else
            return rChild;
    }

    function percUp(uint256[] storage heapList, uint256 i)
        private
    {
        uint256 position = i;
        uint256 value = heapList[i];

        // continue to percolate up while smaller than the parent
        while (i != 0 && value < heapList[i.sub(1).div(2)]) {
            heapList[i] = heapList[i.sub(1).div(2)];
            i = i.sub(1).div(2);
        }

        // place the value in the correct parent
        if (position != i) heapList[i] = value;
    }

    function percDown(uint256[] storage heapList, uint256 i)
        private
    {
        uint position = i;
        uint value = heapList[i];

        // continue to percolate down while larger than the child
        uint child = minChild(heapList, i);
        while(child < heapList.length && value > heapList[child]) {
            heapList[i] = heapList[child];
            i = child;
            child = minChild(heapList, i);
        }

        // place value in the correct child
        if (position != i) heapList[i] = value;
    }

    function currentSize(uint256[] storage heapList)
        internal
        view
        returns (uint256)
    {
        return heapList.length;
    }
}