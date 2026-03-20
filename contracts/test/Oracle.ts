import { expect } from "chai";
import { ethers } from "hardhat";
import { Oracle } from "../typechain-types";
import { SignerWithAddress } from "@nomicfoundation/hardhat-ethers/signers";

describe("Oracle", function () {
  let oracle: Oracle;
  let owner: SignerWithAddress;
  let oracleSigner: SignerWithAddress;
  let otherAccount: SignerWithAddress;

  beforeEach(async function () {
    [owner, oracleSigner, otherAccount] = await ethers.getSigners();
    const OracleFactory = await ethers.getContractFactory("Oracle");
    oracle = await OracleFactory.deploy(oracleSigner.address) as unknown as Oracle;
  });

  it("Should emit OracleRequest on requestPrice", async function () {
    const symbol = "BTC";
    await expect(oracle.requestPrice(symbol))
      .to.emit(oracle, "OracleRequest")
      .withArgs(ethers.AnyValue, symbol);
  });

  it("Should verify signature and update price on fulfill", async function () {
    const symbol = "BTC";
    const tx = await oracle.requestPrice(symbol);
    const receipt = await tx.wait();
    
    // Extract requestId from event
    const event = receipt?.logs.find((log) => {
      try {
        return oracle.interface.parseLog(log)?.name === "OracleRequest";
      } catch (e) {
        return false;
      }
    });
    
    expect(event).to.not.be.undefined;
    const parsedEvent = oracle.interface.parseLog(event as any);
    const requestId = parsedEvent!.args[0];

    const price = 65000;

    // the off-chain signer logic:
    // Message Hash = keccak256(abi.encodePacked(requestId, symbol, price))
    const messageHash = ethers.solidityPackedKeccak256(
      ["bytes32", "string", "uint256"],
      [requestId, symbol, price]
    );

    // Ethers `signMessage` automatically applies the `\x19Ethereum Signed Message:\n32` prefix.
    const messageBytes = ethers.getBytes(messageHash);
    const signature = await oracleSigner.signMessage(messageBytes);

    // Call fulfill
    await expect(oracle.fulfill(requestId, symbol, price, signature))
      .to.emit(oracle, "PriceUpdated")
      .withArgs(symbol, price, ethers.AnyValue);

    // Check state
    const latestPrice = await oracle.latestPrices(symbol);
    expect(latestPrice).to.equal(price);
  });

  it("Should reject invalid signature", async function () {
    const symbol = "ETH";
    const tx = await oracle.requestPrice(symbol);
    const receipt = await tx.wait();
    const event = receipt?.logs.find((log) => {
      try { return oracle.interface.parseLog(log)?.name === "OracleRequest"; } catch(e) { return false; }
    });
    const parsedEvent = oracle.interface.parseLog(event as any);
    const requestId = parsedEvent!.args[0];

    const price = 3000;
    const messageHash = ethers.solidityPackedKeccak256(
      ["bytes32", "string", "uint256"],
      [requestId, symbol, price]
    );
    const messageBytes = ethers.getBytes(messageHash);
    
    // Signed by the WRONG account
    const signature = await otherAccount.signMessage(messageBytes);

    await expect(
      oracle.fulfill(requestId, symbol, price, signature)
    ).to.be.revertedWith("Invalid signature");
  });
});
