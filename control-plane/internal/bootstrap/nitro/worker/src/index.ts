/**
 * Nitro Deployer Worker
 *
 * Node.js package for deploying Arbitrum Nitro chains using POPSigner
 * for secure remote signing via mTLS.
 *
 * @packageDocumentation
 */

// POPSigner Account
export {
  createPOPSignerAccount,
  createPOPSignerAccountFromFiles,
} from './popsigner-account';

export type {
  POPSignerAccount,
  TypedDataInput,
} from './popsigner-account';

// Deployment
export {
  deployOrbitChain,
  parseConfig,
  validateConfig,
} from './deploy';

// Types - POPSigner
export type {
  POPSignerConfig,
  JSONRPCRequest,
  JSONRPCResponse,
  JSONRPCError,
  TransactionParams,
  AccessListItem,
  TypedDataDomain,
} from './types';

// Types - Deployment
export type {
  NitroDeploymentConfig,
  DeploymentResult,
  CoreContracts,
  DataAvailabilityType,
} from './types';

// Errors
export {
  POPSignerError,
  MTLSConfigError,
  DeploymentConfigError,
  DeploymentError,
} from './types';
