import { expect } from "chai";
import { ethers } from "hardhat";
import { OracleRegistry } from "../typechain-types";
import { SignerWithAddress } from "@nomicfoundation/hardhat-ethers/signers";

describe("OracleRegistry", function () {
  let registry: OracleRegistry;
  let owner: SignerWithAddress;
  let treasury: SignerWithAddress;
  let node1: SignerWithAddress;
  let node2: SignerWithAddress;
  let node3: SignerWithAddress;
  let client: SignerWithAddress;

  const MIN_STAKE = ethers.parseEther("1");

  beforeEach(async function () {
    [owner, treasury, node1, node2, node3, client] = await ethers.getSigners();
    const RegistryFactory = await ethers.getContractFactory("OracleRegistry");
    registry = await RegistryFactory.deploy(treasury.address) as unknown as OracleRegistry;
  });

  it("Should register nodes", async function () {
    await expect(registry.connect(node1).registerNode({ value: MIN_STAKE }))
      .to.emit(registry, "NodeRegistered")
      .withArgs(node1.address, MIN_STAKE);
    
    const nodeData = await registry.nodes(node1.address);
    expect(nodeData.isRegistered).to.be.true;
    expect(nodeData.stakeAmount).to.equal(MIN_STAKE);
  });

  it("Should request data", async function () {
    const bounty = ethers.parseEther("0.01");
    // Ensure we send value
    await expect(registry.connect(client).requestData("ETH", { value: bounty }))
      .to.emit(registry, "DataRequested");
  });

  it("Should fulfill request if caller is aggregator", async function () {
    // Register nodes
    await registry.connect(node1).registerNode({ value: MIN_STAKE });
    await registry.connect(node2).registerNode({ value: MIN_STAKE });
    await registry.connect(node3).registerNode({ value: MIN_STAKE });

    // Client requests data
    const bounty = ethers.parseEther("0.3");
    const tx = await registry.connect(client).requestData("ETH", { value: bounty });
    const receipt = await tx.wait();
    
    // Get ReqId
    const event = receipt?.logs.find((log) => {
      try { return registry.interface.parseLog(log)?.name === "DataRequested"; } catch(e) { return false; }
    });
    const parsedEvent = registry.interface.parseLog(event as any);
    const reqId = parsedEvent!.args[0];

    // Get assigned aggregator
    const aggregatorAddress = await registry.getAggregator(reqId);
    
    // Find which signer is the aggregator
    const signers = [node1, node2, node3];
    const aggregatorSigner = signers.find(s => s.address === aggregatorAddress);
    expect(aggregatorSigner).to.not.be.undefined;

    const consensusPrice = 3000;
    const honestNodes = [node1.address, node2.address, node3.address];
    const slashedNodes: string[] = [];

    // Fulfill
    await expect(
      registry.connect(aggregatorSigner!).fulfillRequest(reqId, consensusPrice, honestNodes, slashedNodes)
    ).to.emit(registry, "RequestFulfilled").withArgs(reqId, consensusPrice, aggregatorAddress);
    
    // Check request resolved
    const reqData = await registry.requests(reqId);
    expect(reqData.resolved).to.be.true;

    // Check rewards distributed (trust score should be 101 now)
    const n1 = await registry.nodes(node1.address);
    expect(n1.trustScore).to.equal(101n);
  });
});
