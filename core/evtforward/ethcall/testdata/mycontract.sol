// SPDX-License-Identifier: MIT

pragma solidity >0.7.0 <0.9.0;

/**
 * @title Storage
 * @dev store or retrieve variable value
 */

contract MyContract {
    function get_uint256(uint256 input) public pure returns (uint256) {
        return input;
        // return 52;
    }

    struct Person {
        string name;
        uint16 age;
    }

    function testy(
        int64,
        uint256,
        string calldata,
        bool,
        address,
        int[] calldata,
        Person calldata
    ) public pure returns (uint256) {
        // return input;
        return 52;
    }
}
