// SPDX-License-Identifier: MIT

pragma solidity >0.7.0 <0.9.0;

/**
 * @title Storage
 * @dev store or retrieve variable value
 */

contract MyContract {
    function get_uint256(uint256 input) public pure returns (uint256) {
        return input;
    }

    struct Person {
        string name;
        uint16 age;
    }

    function testy1(
        int64 in1,
        uint256 in2,
        string calldata in3,
        bool in4,
        address in5
    ) public pure returns (int64, uint256, string calldata, bool, address) {
        return (in1, in2, in3, in4, in5);
    }

    // get stack to deep if you have too many locals
    function testy2(
        int[] calldata in6,
        Person calldata in7
    ) public pure returns (int[] calldata, Person calldata) {
        return (in6, in7);
    }
}
