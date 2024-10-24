// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

abstract contract Ass {
  function getSeiAddr(
      address addr
  ) public virtual view returns (string memory response);
}
