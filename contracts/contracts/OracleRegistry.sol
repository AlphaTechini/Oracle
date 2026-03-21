// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/**
 * @title OracleRegistry
 * @dev Decentralized Oracle Network (DON) Registry and Settlement Contract.
 * Optimized with Solidity 0.8.24 Transient Storage (EIP-1153) and Custom Errors.
 */
contract OracleRegistry {
    // --- Constants ---
    uint256 public constant MIN_STAKE = 1 ether;
    uint256 public constant MIN_CONSENSUS_NODES = 3;
    uint256 public constant TRUST_SCORE_INITIAL = 100;
    uint256 public constant TRUST_SCORE_INCREMENT = 1;
    uint256 public constant TRUST_SCORE_DECREMENT = 10;
    uint256 public constant SLASH_PERCENTAGE_BPS = 1000; // 10% (1000 basis points)

    // Transient Storage Slot for Reentrancy Guard
    bytes32 private constant REENTRANCY_GUARD_SLOT = keccak256("oracle.reentrancy.guard");

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

    // --- Custom Errors ---
    error OnlyOwner();
    error InsufficientStake();
    error AlreadyRegistered();
    error BountyRequired();
    error RequestAlreadyResolved();
    error InsufficientConsensus();
    error UnauthorizedAggregator(address caller, address expected);
    error NodeNotActive(address node);
    error ReentrantCall();

    // --- Modifiers ---
    modifier onlyOwner() {
        if (msg.sender != owner) revert OnlyOwner();
        _;
    }

    /**
     * @dev Gas-efficient reentrancy guard using EIP-1153 Transient Storage (Cost: 100 gas vs 20,000 gas)
     */
    modifier nonReentrant() {
        uint256 guardStatus;
        bytes32 slot = REENTRANCY_GUARD_SLOT;
        assembly {
            guardStatus := tload(slot)
        }
        if (guardStatus == 1) revert ReentrantCall();
        
        assembly {
            tstore(slot, 1)
        }
        _;
        assembly {
            tstore(slot, 0)
        }
    }

    // --- Constructor ---
    constructor(address _treasury) {
        owner = msg.sender;
        protocolTreasury = _treasury;
    }

    // --- Core Functions ---

    /**
     * @dev Register a new node by staking MIN_STAKE.
     */
    function registerNode() external payable nonReentrant {
        if (msg.value < MIN_STAKE) revert InsufficientStake();
        if (nodes[msg.sender].isRegistered) revert AlreadyRegistered();

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
    function requestData(string calldata symbol) external payable nonReentrant returns (bytes32) {
        if (msg.value == 0) revert BountyRequired();
        
        bytes32 reqId = keccak256(abi.encodePacked(symbol, msg.sender, block.timestamp));
        
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
     * @dev Fulfill a request with aggregated data.
     * Gas Optimization: Instead of looping `ecrecover` for every signature, the contract
     * verifies that the caller is the deterministically assigned Aggregator for this round.
     * The Aggregator is trusted to have verified the BLS/Threshold consensus off-chain.
     */
    function fulfillRequest(
        bytes32 reqId,
        uint256 consensusPrice,
        address[] calldata honestNodes,
        address[] calldata slashedNodes
    ) external nonReentrant {
        Request storage req = requests[reqId];
        if (req.resolved) revert RequestAlreadyResolved();
        
        address assignedAggregator = getAggregator(reqId);
        if (msg.sender != assignedAggregator) revert UnauthorizedAggregator(msg.sender, assignedAggregator);
        if (honestNodes.length < MIN_CONSENSUS_NODES) revert InsufficientConsensus();

        // 1. Distribute Rewards to Honest Nodes
        uint256 rewardPerNode = req.bountyFee / honestNodes.length;
        for (uint256 i = 0; i < honestNodes.length; i++) {
            address nodeAddr = honestNodes[i];
            if (!nodes[nodeAddr].isActive) revert NodeNotActive(nodeAddr);
            
            nodes[nodeAddr].trustScore += TRUST_SCORE_INCREMENT;
            payable(nodeAddr).transfer(rewardPerNode);
        }

        // 2. Slash Malicious/Inaccurate Nodes
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

    /**
     * @dev Deterministically pick the next aggregator for a request based on length and reqId.
     */
    function getAggregator(bytes32 reqId) public view returns (address) {
        if (registeredNodes.length == 0) return address(0);
        uint256 index = uint256(keccak256(abi.encodePacked(reqId))) % registeredNodes.length;
        return registeredNodes[index];
    }
}
