#!/bin/bash

abigen --abi ERC20_Bridge_Logic_Restricted.abi --pkg erc20_bridge_logic_restricted  --out ./erc20_bridge_logic_restricted/erc20_bridge_logic_restricted.go
abigen --abi MultisigControl.abi --pkg multisig_control  --out ./multisig_control/multisig_control.go
abigen --abi ERC20.abi --pkg erc20  --out ./erc20/erc20.go
