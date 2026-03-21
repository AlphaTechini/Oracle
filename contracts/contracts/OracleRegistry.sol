// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title OracleRegistry
 * @dev Decentralized Oracle Network (DON) Registry and Settlement Contract.
 * Handles node registration, staking, data requests, and fulfillment with consensus.
 */
contract OracleRegistry {
    // --- Constants ---
    uint256 public constant MIN_STAKE = 1 ether;
    uint256 public constant MIN_CONSENSUS_NODES = 3;
    uint256 public constant DEVIATION_THRESHOLD_BPS = 20; // 0.2% (20 basis points)
    uint256 public constant TRUST_SCORE_INITIAL = 100;
    uint256 public constant TRUST_SCORE_INCREMENT = 1;
    uint256 public constant TRUST_SCORE_DECREMENT = 10;
    uint256 public constant SLASH_PERCENTAGE_BPS = 1000; // 10% (1000 basis points)

    // --- Structs ---
    struct Node {
        uint256 stakeAmount;
        uint256 trustScore;
        bool isRegistered;
        bool isActive;
    }

    struct Request {
        address client;
        string symbol;
        uint256 bountyFee;
        bool resolved;
    }

    // --- State Variables ---
    address public owner;
    address public protocolTreasury;
    
    mapping(address => Node) public nodes;
    address[] public registeredNodes;
    mapping(bytes32 => Request) public requests;
    
    // --- Events ---
    event NodeRegistered(address indexed node, uint256 stake);
    event DataRequested(bytes32 indexed reqId, string symbol, uint256 bounty);
    event RequestFulfilled(bytes32 indexed reqId, uint256 consensusPrice, address aggregator);
    event NodeSlashed(address indexed node, uint256 amountSlashed);

    // --- Constructor ---
    constructor(address _treasury) {
        owner = msg.sender;
        protocolTreasury = _treasury;
    }

    // --- Modifiers ---
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }

    // --- Core Functions ---

    /**
     * @dev Register a new node by staking MIN_STAKE.
     */
    function registerNode() external payable {
        require(msg.value >= MIN_STAKE, "Insufficient stake");
        require(!nodes[msg.sender].isRegistered, "Already registered");

        nodes[msg.sender] = Node({
            stakeAmount: msg.value,
            trustScore: TRUST_SCORE_INITIAL,
            isRegistered: true,
            isActive: true
        });

        registeredNodes.push(msg.sender);
        emit NodeRegistered(msg.sender, msg.value);
    }

    /**
     * @dev Request data for a specific symbol. Client must send bountyFee as msg.value.
     */
    function requestData(string calldata symbol) external payable returns (bytes32) {
        require(msg.value > 0, "Bounty required");
        
        bytes32 reqId = keccak256(abi.encodePacked(symbol, msg.sender, block.timestamp, block.prevrandao));
        
        requests[reqId] = Request({
            client: msg.sender,
            symbol: symbol,
            bountyFee: msg.value,
            resolved: false
        });

        emit DataRequested(reqId, symbol, msg.value);
        return reqId;
    }

    /**
     * @dev Fulfill a request with aggregated data and signatures.
     * Can be called by any node, but rotation logic off-chain ensures order.
     */
    function fulfillRequest(
        bytes32 reqId,
        uint256 consensusPrice,
        address[] calldata honestNodes,
        address[] calldata slashedNodes,
        bytes[] calldata signatures
    ) external {
        Request storage req = requests[reqId];
        require(!req.resolved, "Already resolved");
        require(honestNodes.length >= MIN_CONSENSUS_NODES, "Insufficient consensus");
        require(honestNodes.length == signatures.length, "Mismatch signatures");

        // 1. Verify Signatures
        bytes32 messageHash = keccak256(abi.encodePacked(reqId, consensusPrice));
        bytes32 ethSignedMessageHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash));

        for (uint256 i = 0; i < honestNodes.length; i++) {
            address nodeAddr = honestNodes[i];
            require(nodes[nodeAddr].isActive, "Node not active");
            
            // Reconstruct and verify signature
            address signer = recoverSigner(ethSignedMessageHash, signatures[i]);
            require(signer == nodeAddr, "Invalid signature");
            
            // Reward Honest Node
            nodes[nodeAddr].trustScore += TRUST_SCORE_INCREMENT;
        }

        // 2. Distribute Rewards
        uint256 rewardPerNode = req.bountyFee / honestNodes.length;
        for (uint256 i = 0; i < honestNodes.length; i++) {
            payable(honestNodes[i]).transfer(rewardPerNode);
        }

        // 3. Slash Malicious/Inaccurate Nodes
        for (uint256 i = 0; i < slashedNodes.length; i++) {
            address nodeAddr = slashedNodes[i];
            if (nodes[nodeAddr].isRegistered) {
                uint256 slashAmount = (nodes[nodeAddr].stakeAmount * SLASH_PERCENTAGE_BPS) / 10000;
                nodes[nodeAddr].stakeAmount -= slashAmount;
                nodes[nodeAddr].trustScore = nodes[nodeAddr].trustScore > TRUST_SCORE_DECREMENT 
                    ? nodes[nodeAddr].trustScore - TRUST_SCORE_DECREMENT 
                    : 0;
                
                payable(protocolTreasury).transfer(slashAmount);
                emit NodeSlashed(nodeAddr, slashAmount);
            }
        }

        req.resolved = true;
        emit RequestFulfilled(reqId, consensusPrice, msg.sender);
    }

    // --- Helper Functions ---

    function recoverSigner(bytes32 _ethSignedMessageHash, bytes memory _signature) public pure returns (address) {
        (bytes32 r, bytes32 s, uint8 v) = splitSignature(_signature);
        return ecrecover(_ethSignedMessageHash, v, r, s);
    }

    function splitSignature(bytes memory sig) public pure returns (bytes32 r, bytes32 s, uint8 v) {
        require(sig.length == 65, "Invalid signature length");
        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
    }

    /**
     * @dev Deterministically pick the next aggregator for a request.
     */
    function getAggregator(bytes32 reqId) public view returns (address) {
        if (registeredNodes.length == 0) return address(0);
        uint256 index = uint256(keccak256(abi.encodePacked(reqId))) % registeredNodes.length;
        return registeredNodes[index];
    }
}
