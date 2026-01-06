/**
 * Nitro Orbit Chain Deployment
 *
 * Deploys a new Arbitrum Nitro/Orbit chain using the orbit-sdk.
 * Uses POPSigner for mTLS-authenticated transaction signing.
 *
 * @module deploy
 */

import {
  createPublicClient,
  createWalletClient,
  http,
  defineChain,
  encodeFunctionData,
  type Address,
  type Hex,
  type PublicClient,
  type WalletClient,
  type Transport,
  type Chain,
} from 'viem';
import { arbitrum, arbitrumSepolia } from 'viem/chains';
import {
  createRollupPrepareDeploymentParamsConfig,
  createRollupPrepareTransactionRequest,
  prepareChainConfig,
  createRollupPrepareTransactionReceipt,
} from '@arbitrum/orbit-sdk';

import { createPOPSignerAccount, type POPSignerAccount } from './popsigner-account';
import {
  type NitroDeploymentConfig,
  type DeploymentResult,
  type CoreContracts,
  DeploymentConfigError,
  DeploymentError,
} from './types';

/**
 * Zero address constant.
 */
const ZERO_ADDRESS = '0x0000000000000000000000000000000000000000' as Address;

/**
 * SequencerInbox ABI for batch poster management.
 * We only need setIsBatchPoster and isBatchPoster functions.
 */
const SEQUENCER_INBOX_ABI = [
  {
    inputs: [
      { name: 'addr', type: 'address' },
      { name: 'isBatchPoster_', type: 'bool' },
    ],
    name: 'setIsBatchPoster',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [{ name: '', type: 'address' }],
    name: 'isBatchPoster',
    outputs: [{ name: '', type: 'bool' }],
    stateMutability: 'view',
    type: 'function',
  },
] as const;

/**
 * UpgradeExecutor ABI for executing privileged calls.
 * setIsBatchPoster must be called through UpgradeExecutor as it's the rollup owner.
 */
const UPGRADE_EXECUTOR_ABI = [
  {
    inputs: [
      { name: 'upgrade', type: 'address' },
      { name: 'upgradeCallData', type: 'bytes' },
    ],
    name: 'executeCall',
    outputs: [],
    stateMutability: 'payable',
    type: 'function',
  },
] as const;

/**
 * Default deployment parameters.
 */
const DEFAULTS = {
  confirmPeriodBlocks: 45818, // ~1 week on Ethereum
  extraChallengeTimeBlocks: 0,
  maxDataSize: 117964,
  deployFactoriesToL2: true,
};

/**
 * Logs a message to stderr (for debugging, not captured by Go wrapper).
 */
function log(message: string): void {
  console.error(`[nitro-deployer] ${message}`);
}

/**
 * Validates the deployment configuration.
 * @throws {DeploymentConfigError} If configuration is invalid.
 */
export function validateConfig(config: NitroDeploymentConfig): void {
  const required: (keyof NitroDeploymentConfig)[] = [
    'chainId',
    'chainName',
    'parentChainId',
    'parentChainRpc',
    'owner',
    'batchPosters',
    'validators',
    'stakeToken',
    'baseStake',
    // dataAvailability is optional - defaults to 'celestia'
    'popsignerEndpoint',
    'clientCert',
    'clientKey',
  ];

  for (const field of required) {
    if (config[field] === undefined || config[field] === null) {
      throw new DeploymentConfigError(`Missing required field: ${field}`, field);
    }
  }

  // Validate chainId
  if (config.chainId <= 0) {
    throw new DeploymentConfigError('chainId must be positive', 'chainId');
  }

  // Validate arrays have at least one element
  if (config.batchPosters.length === 0) {
    throw new DeploymentConfigError('At least one batch poster required', 'batchPosters');
  }

  if (config.validators.length === 0) {
    throw new DeploymentConfigError('At least one validator required', 'validators');
  }

  // Validate baseStake is a valid bigint string
  try {
    const stake = BigInt(config.baseStake);
    if (stake <= 0n) {
      throw new DeploymentConfigError('baseStake must be positive', 'baseStake');
    }
  } catch (error) {
    // Re-throw our own errors
    if (error instanceof DeploymentConfigError) {
      throw error;
    }
    throw new DeploymentConfigError('baseStake must be a valid integer string', 'baseStake');
  }

  // Validate dataAvailability - default to celestia if not provided or invalid
  // POPSigner deployments use Celestia DA by default
  if (config.dataAvailability && !['rollup', 'anytrust', 'celestia'].includes(config.dataAvailability)) {
    throw new DeploymentConfigError(
      'dataAvailability must be "celestia", "rollup", or "anytrust"',
      'dataAvailability',
    );
  }

  // Validate POPSigner endpoint
  if (!config.popsignerEndpoint.startsWith('https://')) {
    throw new DeploymentConfigError(
      'popsignerEndpoint must use HTTPS',
      'popsignerEndpoint',
    );
  }
}

/**
 * Gets the Viem chain definition for a given chain ID.
 */
function getParentChain(chainId: number, rpcUrl: string): Chain {
  switch (chainId) {
    case 42161:
      return arbitrum;
    case 421614:
      return arbitrumSepolia;
    default:
      // For other chains, create a custom chain config
      return defineChain({
        id: chainId,
        name: `Chain ${chainId}`,
        network: `chain-${chainId}`,
        nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
        rpcUrls: {
          default: { http: [rpcUrl] },
          public: { http: [rpcUrl] },
        },
      });
  }
}

/**
 * Deploys a new Nitro/Orbit chain.
 *
 * This is an atomic operation - the entire chain is deployed in a single transaction.
 * Uses the POPSigner Viem account for mTLS-authenticated signing.
 *
 * @param config - Deployment configuration
 * @returns Deployment result with contract addresses or error
 */
export async function deployOrbitChain(
  config: NitroDeploymentConfig,
): Promise<DeploymentResult> {
  try {
    // Validate configuration
    validateConfig(config);
    
    log(`Deploying Orbit chain ${config.chainName} (ID: ${config.chainId})`);
    log(`Parent chain: ${config.parentChainId}`);
    log(`Data availability: ${config.dataAvailability}`);

    // Get parent chain definition
    const parentChain = getParentChain(config.parentChainId, config.parentChainRpc);

    // Create POPSigner account for signing
    const account = createPOPSignerAccount({
      endpoint: config.popsignerEndpoint,
      address: config.owner,
      clientCert: config.clientCert,
      clientKey: config.clientKey,
      caCert: config.caCert,
      timeout: 60000, // 60 second timeout for deployment
    });

    log(`Using deployer address: ${account.address}`);

    // Create public client for reading chain state
    const publicClient = createPublicClient({
      chain: parentChain,
      transport: http(config.parentChainRpc),
    });

    // Check deployer balance
    const balance = await publicClient.getBalance({ address: config.owner });
    const balanceEth = Number(balance) / 1e18;
    log(`Deployer balance: ${balanceEth.toFixed(6)} ETH`);
    
    if (balance === 0n) {
      throw new DeploymentConfigError('Deployer address has no ETH balance', 'owner');
    }
    if (balance < 100000000000000000n) { // 0.1 ETH minimum
      log(`WARNING: Low balance. Nitro deployment typically requires 0.5-1 ETH`);
    }

    // Prepare chain configuration
    // DataAvailabilityCommittee:
    //   false = Rollup mode OR External DA provider (Celestia)
    //   true  = AnyTrust DAC mode
    // For Celestia, we use external-provider flags with DAC=false
    const dataAvailability = config.dataAvailability || 'celestia';
    const chainConfig = prepareChainConfig({
      chainId: config.chainId,
      arbitrum: {
        InitialChainOwner: config.owner,
        DataAvailabilityCommittee: dataAvailability === 'anytrust',
      },
    });

    log('Chain config prepared');

    // Prepare deployment parameters using orbit-sdk
    const deploymentConfig = await createRollupPrepareDeploymentParamsConfig(publicClient, {
      chainId: BigInt(config.chainId),
      owner: config.owner,
      chainConfig,
    });

    log('Deployment config prepared');

    // Prepare the deployment transaction request
    const txRequest = await createRollupPrepareTransactionRequest({
      params: {
        config: deploymentConfig,
        batchPosters: config.batchPosters,
        validators: config.validators,
        nativeToken: config.nativeToken ?? ZERO_ADDRESS,
        deployFactoriesToL2: config.deployFactoriesToL2 ?? DEFAULTS.deployFactoriesToL2,
        maxDataSize: BigInt(config.maxDataSize ?? DEFAULTS.maxDataSize),
      },
      account: account.address,
      publicClient,
    });

    log('Transaction request prepared');
    
    // Get current gas prices and add buffer for faster inclusion
    // This is critical for good UX - transactions with low gas get stuck
    const feeData = await publicClient.estimateFeesPerGas();
    const baseFee = feeData.maxFeePerGas ?? 0n;
    const priorityFee = feeData.maxPriorityFeePerGas ?? 0n;
    
    // Add 50% buffer to base fee and use at least 2 Gwei priority fee
    const minPriorityFee = 2000000000n; // 2 Gwei minimum
    const boostedPriorityFee = priorityFee > minPriorityFee ? priorityFee : minPriorityFee;
    
    // Calculate max fee: at least 1.5x base fee, but MUST be >= priority fee
    // EIP-1559 requires maxFeePerGas >= maxPriorityFeePerGas
    const calculatedMaxFee = (baseFee * 150n) / 100n; // 1.5x base fee
    const boostedMaxFee = calculatedMaxFee > boostedPriorityFee 
      ? calculatedMaxFee 
      : boostedPriorityFee + (baseFee / 2n); // priority + some headroom for base fee
    
    log(`Current base fee: ${Number(baseFee) / 1e9} Gwei`);
    log(`Boosted max fee: ${Number(boostedMaxFee) / 1e9} Gwei`);
    log(`Priority fee: ${Number(boostedPriorityFee) / 1e9} Gwei`);
    
    log('Sending deployment transaction...');

    // Create wallet client for sending transactions
    // We need to cast the account to satisfy viem's types
    const walletClient = createWalletClient({
      account: account as unknown as `0x${string}`,
      chain: parentChain,
      transport: http(config.parentChainRpc),
    });

    // Send the deployment transaction using the POPSigner account
    // We manually sign and send since our account type doesn't match viem's exactly
    // Override gas prices with boosted values for faster inclusion
    const txWithGas = {
      ...txRequest,
      chainId: parentChain.id,
      maxFeePerGas: boostedMaxFee,
      maxPriorityFeePerGas: boostedPriorityFee,
      type: 'eip1559' as const,
    };
    const signedTx = await account.signTransaction(txWithGas);

    // Broadcast the signed transaction
    const txHash = await publicClient.sendRawTransaction({
      serializedTransaction: signedTx,
    });

    log(`Transaction submitted: ${txHash}`);
    log('Waiting for confirmation...');

    // Wait for transaction receipt with retry logic
    // Some RPC providers have latency in returning receipts
    let receipt;
    let attempts = 0;
    const maxAttempts = 30; // 30 attempts * 10 seconds = 5 minutes max
    
    while (attempts < maxAttempts) {
      try {
        receipt = await publicClient.waitForTransactionReceipt({
          hash: txHash,
          confirmations: 1, // Wait for 1 confirmation first
          timeout: 60_000, // 1 minute per attempt
          pollingInterval: 2_000, // Poll every 2 seconds
        });
        break; // Success, exit loop
      } catch (receiptError: unknown) {
        attempts++;
        const errorMessage = receiptError instanceof Error ? receiptError.message : String(receiptError);
        
        // If it's not a "receipt not found" error, rethrow
        if (!errorMessage.includes('could not be found')) {
          throw receiptError;
        }
        
        log(`Receipt not yet available (attempt ${attempts}/${maxAttempts}), retrying...`);
        
        if (attempts >= maxAttempts) {
          // Final attempt - provide helpful message with tx hash
          throw new DeploymentError(
            `Transaction submitted but receipt not found after ${maxAttempts} attempts. ` +
            `Check transaction status on block explorer: https://sepolia.etherscan.io/tx/${txHash}`,
            txHash
          );
        }
        
        // Wait before retry
        await new Promise(resolve => setTimeout(resolve, 10_000));
      }
    }
    
    if (!receipt) {
      throw new DeploymentError('Failed to get transaction receipt', txHash);
    }

    log(`Transaction confirmed in block ${receipt.blockNumber}`);

    // Check transaction status
    if (receipt.status !== 'success') {
      // Try to get revert reason by simulating the transaction
      let revertReason = 'Unknown reason';
      try {
        // Attempt to call the transaction to get revert reason
        await publicClient.call({
          to: txRequest.to,
          data: txRequest.data,
          account: txRequest.from,
          gas: txRequest.gas,
          value: txRequest.value,
        });
      } catch (callError: unknown) {
        if (callError instanceof Error) {
          revertReason = callError.message;
        }
      }
      log(`Transaction reverted. Reason: ${revertReason}`);
      log(`Gas used: ${receipt.gasUsed}`);
      throw new DeploymentError(`Transaction reverted: ${revertReason}`, txHash);
    }

    // Parse contract addresses from receipt using orbit-sdk helper
    const txReceipt = createRollupPrepareTransactionReceipt(receipt);
    const coreContracts = txReceipt.getCoreContracts();

    log('Core contracts deployed!');
    log(`Rollup address: ${coreContracts.rollup}`);
    log(`SequencerInbox address: ${coreContracts.sequencerInbox}`);

    // =========================================================================
    // CRITICAL: Whitelist batch posters on SequencerInbox via UpgradeExecutor
    // The RollupCreator does NOT automatically whitelist batch posters!
    // Without this, batch poster will fail with NotBatchPoster() error
    // 
    // The SequencerInbox is owned by the UpgradeExecutor, so we must call
    // setIsBatchPoster through the UpgradeExecutor.executeCall() function.
    // The deployer address has EXECUTOR_ROLE on the UpgradeExecutor.
    // =========================================================================
    if (config.batchPosters && config.batchPosters.length > 0) {
      log(`Whitelisting ${config.batchPosters.length} batch poster(s) via UpgradeExecutor...`);
      log(`  UpgradeExecutor: ${coreContracts.upgradeExecutor}`);
      log(`  SequencerInbox: ${coreContracts.sequencerInbox}`);
      
      for (const batchPoster of config.batchPosters) {
        log(`  Whitelisting batch poster: ${batchPoster}`);
        
        // Check if already whitelisted (shouldn't be, but check anyway)
        const isAlreadyWhitelisted = await publicClient.readContract({
          address: coreContracts.sequencerInbox,
          abi: SEQUENCER_INBOX_ABI,
          functionName: 'isBatchPoster',
          args: [batchPoster],
        });
        
        if (isAlreadyWhitelisted) {
          log(`  Already whitelisted: ${batchPoster}`);
          continue;
        }
        
        // Encode the inner call: SequencerInbox.setIsBatchPoster(batchPoster, true)
        const innerCallData = encodeFunctionData({
          abi: SEQUENCER_INBOX_ABI,
          functionName: 'setIsBatchPoster',
          args: [batchPoster, true],
        });
        
        // Encode the outer call: UpgradeExecutor.executeCall(sequencerInbox, innerCallData)
        const outerCallData = encodeFunctionData({
          abi: UPGRADE_EXECUTOR_ABI,
          functionName: 'executeCall',
          args: [coreContracts.sequencerInbox, innerCallData],
        });
        
        // Prepare the transaction to UpgradeExecutor
        const setBatchPosterData = {
          to: coreContracts.upgradeExecutor,
          data: outerCallData,
          chainId: parentChain.id,
          maxFeePerGas: boostedMaxFee,
          maxPriorityFeePerGas: boostedPriorityFee,
          type: 'eip1559' as const,
        };
        
        // Estimate gas for the transaction
        const gasEstimate = await publicClient.estimateGas({
          ...setBatchPosterData,
          account: account.address,
        });
        
        const setBatchPosterTx = {
          ...setBatchPosterData,
          gas: (gasEstimate * 120n) / 100n, // 20% buffer
          nonce: await publicClient.getTransactionCount({ address: account.address }),
        };
        
        const signedSetBatchPosterTx = await account.signTransaction(setBatchPosterTx);
        const setBatchPosterHash = await publicClient.sendRawTransaction({
          serializedTransaction: signedSetBatchPosterTx,
        });
        
        log(`  UpgradeExecutor.executeCall tx submitted: ${setBatchPosterHash}`);
        
        // Wait for confirmation
        const setBatchPosterReceipt = await publicClient.waitForTransactionReceipt({
          hash: setBatchPosterHash,
          confirmations: 1,
          timeout: 60_000,
        });
        
        if (setBatchPosterReceipt.status !== 'success') {
          throw new DeploymentError(
            `Failed to whitelist batch poster ${batchPoster} via UpgradeExecutor`,
            setBatchPosterHash
          );
        }
        
        // Verify it worked
        const isNowWhitelisted = await publicClient.readContract({
          address: coreContracts.sequencerInbox,
          abi: SEQUENCER_INBOX_ABI,
          functionName: 'isBatchPoster',
          args: [batchPoster],
        });
        
        if (!isNowWhitelisted) {
          throw new DeploymentError(
            `Batch poster ${batchPoster} not whitelisted after transaction - check UpgradeExecutor permissions`,
            setBatchPosterHash
          );
        }
        
        log(`  Batch poster whitelisted successfully: ${batchPoster}`);
      }
      
      log('All batch posters whitelisted!');
    }

    log('Deployment successful!');
    log(`Inbox address: ${coreContracts.inbox}`);

    return {
      success: true,
      coreContracts: {
        rollup: coreContracts.rollup,
        inbox: coreContracts.inbox,
        outbox: coreContracts.outbox,
        bridge: coreContracts.bridge,
        sequencerInbox: coreContracts.sequencerInbox,
        rollupEventInbox: coreContracts.rollupEventInbox,
        challengeManager: coreContracts.challengeManager,
        adminProxy: coreContracts.adminProxy,
        upgradeExecutor: coreContracts.upgradeExecutor,
        validatorWalletCreator: coreContracts.validatorWalletCreator,
        nativeToken: coreContracts.nativeToken ?? ZERO_ADDRESS,
        deployedAtBlockNumber: Number(receipt.blockNumber),
      },
      transactionHash: txHash,
      blockNumber: Number(receipt.blockNumber),
      chainConfig: chainConfig as Record<string, unknown>,
    };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    const txHash = error instanceof DeploymentError ? error.transactionHash : undefined;

    log(`Deployment failed: ${errorMessage}`);

    return {
      success: false,
      error: errorMessage,
      transactionHash: txHash,
    };
  }
}

/**
 * Parses deployment config from a JSON string.
 * @throws {DeploymentConfigError} If JSON is invalid.
 */
export function parseConfig(json: string): NitroDeploymentConfig {
  try {
    const config = JSON.parse(json) as Partial<NitroDeploymentConfig>;
    
    // Apply defaults - POPSigner uses Celestia DA by default
    return {
      confirmPeriodBlocks: DEFAULTS.confirmPeriodBlocks,
      extraChallengeTimeBlocks: DEFAULTS.extraChallengeTimeBlocks,
      maxDataSize: DEFAULTS.maxDataSize,
      deployFactoriesToL2: DEFAULTS.deployFactoriesToL2,
      // Override with provided values
      ...config,
      // Ensure dataAvailability defaults to celestia if not provided
      dataAvailability: config.dataAvailability || 'celestia',
    } as NitroDeploymentConfig;
  } catch (error) {
    throw new DeploymentConfigError(
      `Invalid JSON: ${error instanceof Error ? error.message : String(error)}`,
    );
  }
}
