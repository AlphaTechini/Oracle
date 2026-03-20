import { ethers } from "hardhat";

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contract with account:", deployer.address);

  // We use the deployer as the initial approved oracleSigner for simplicity, 
  // or it could be defined in .env
  const OracleFactory = await ethers.getContractFactory("Oracle");
  const oracle = await OracleFactory.deploy(deployer.address);

  await oracle.waitForDeployment();
  const address = await oracle.getAddress();

  console.log("Oracle deployed to:", address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
