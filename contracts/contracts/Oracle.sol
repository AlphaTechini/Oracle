// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/**
 * @title Web3 DeFi Oracle
 * @dev A high-throughput Oracle that requests prices and verifies off-chain signatures.
 */
contract Oracle {
    address public oracleSigner;

    // symbol => price
    mapping(string => uint256) public latestPrices;
    
    // To prevent replay attacks (simple nonce per symbol, or request ID)
    mapping(bytes32 => bool) public processedRequests;

    event OracleRequest(bytes32 indexed requestId, string symbol);
    event PriceUpdated(string symbol, uint256 price, uint256 timestamp);

    constructor(address _oracleSigner) {
        require(_oracleSigner != address(0), "Invalid signer address");
        oracleSigner = _oracleSigner;
    }

    /**
     * @dev Emit an event requesting the off-chain engine to fetch a price.
     * @param symbol The ticker to fetch (e.g., "BTC", "ETH")
     */
    function requestPrice(string calldata symbol) external returns (bytes32) {
        bytes32 requestId = keccak256(abi.encodePacked(symbol, block.timestamp, msg.sender));
        emit OracleRequest(requestId, symbol);
        return requestId;
    }

    /**
     * @dev Called by the off-chain dispatcher to fulfill a price request.
     * @param requestId The ID emitted in `OracleRequest`.
     * @param symbol The ticker that was requested.
     * @param price The fetched price.
     * @param signature The ECDSA signature from the trusted off-chain signer.
     */
    function fulfill(
        bytes32 requestId,
        string calldata symbol,
        uint256 price,
        bytes calldata signature
    ) external {
        require(!processedRequests[requestId], "Request already processed");

        // Recreate the message hash that was signed
        bytes32 messageHash = keccak256(abi.encodePacked(requestId, symbol, price));
        bytes32 ethSignedMessageHash = _getEthSignedMessageHash(messageHash);

        // Recover the signer address
        address recoveredSigner = _recoverSigner(ethSignedMessageHash, signature);
        require(recoveredSigner == oracleSigner, "Invalid signature");

        // Update state
        processedRequests[requestId] = true;
        latestPrices[symbol] = price;

        emit PriceUpdated(symbol, price, block.timestamp);
    }

    /**
     * @dev Prefix the hash as per ERC-191.
     */
    function _getEthSignedMessageHash(bytes32 _messageHash) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", _messageHash));
    }

    /**
     * @dev Recover the signer from the signature.
     */
    function _recoverSigner(bytes32 _ethSignedMessageHash, bytes memory _signature) internal pure returns (address) {
        require(_signature.length == 65, "Invalid signature length");

        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            r := mload(add(_signature, 32))
            s := mload(add(_signature, 64))
            v := byte(0, mload(add(_signature, 96)))
        }

        return ecrecover(_ethSignedMessageHash, v, r, s);
    }
}
