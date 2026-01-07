package nitro

// NitroNodeConfig represents the full Nitro node configuration.
type NitroNodeConfig struct {
	ParentChain ParentChainConfig `json:"parent-chain"`
	Chain       ChainNodeConfig   `json:"chain"`
	HTTP        HTTPConfig        `json:"http"`
	WS          WSConfig          `json:"ws"`
	Node        NodeSettings      `json:"node"`
	Execution   *ExecutionConfig  `json:"execution,omitempty"`
	Metrics     MetricsConfig     `json:"metrics"`
}

type ParentChainConfig struct {
	Connection ConnectionConfig `json:"connection"`
}

type ConnectionConfig struct {
	URL string `json:"url"`
}

type ChainNodeConfig struct {
	ID        uint64 `json:"id"`
	InfoFiles string `json:"info-files,omitempty"`
}

type HTTPConfig struct {
	Addr       string   `json:"addr"`
	Port       int      `json:"port"`
	VHosts     string   `json:"vhosts"`
	Corsdomain string   `json:"corsdomain"`
	API        []string `json:"api"`
}

type WSConfig struct {
	Addr string   `json:"addr"`
	Port int      `json:"port"`
	API  []string `json:"api"`
}

type NodeSettings struct {
	Sequencer        SequencerConfig   `json:"sequencer"`
	BatchPoster      BatchPosterConfig `json:"batch-poster"`
	Staker           StakerConfig      `json:"staker"`
	DataAvailability *DAConfig         `json:"data-availability,omitempty"`
	DelayedSequencer DelayedSeqConfig  `json:"delayed-sequencer"`
}

type SequencerConfig struct {
	Enable bool `json:"enable"`
}

type BatchPosterConfig struct {
	Enable     bool             `json:"enable"`
	DataPoster DataPosterConfig `json:"data-poster"`
}

type DataPosterConfig struct {
	ExternalSigner ExternalSignerConfig `json:"external-signer"`
}

type ExternalSignerConfig struct {
	URL              string `json:"url"`
	Method           string `json:"method"`
	ClientCert       string `json:"client-cert"`
	ClientPrivateKey string `json:"client-private-key"`
}

type StakerConfig struct {
	Enable     bool             `json:"enable"`
	Strategy   string           `json:"strategy,omitempty"`
	DataPoster DataPosterConfig `json:"data-poster"`
}

type DAConfig struct {
	Enable             bool            `json:"enable"`
	SequencerInboxAddr string          `json:"sequencer-inbox-address"`
	Celestia           *CelestiaConfig `json:"celestia,omitempty"`
}

type CelestiaConfig struct {
	Enable    bool   `json:"enable"`
	ServerURL string `json:"rpc-url"`
}

type DelayedSeqConfig struct {
	Enable bool `json:"enable"`
}

type ExecutionConfig struct {
	ForwardingTarget string `json:"forwarding-target,omitempty"`
}

type MetricsConfig struct {
	Server MetricsServerConfig `json:"server"`
}

type MetricsServerConfig struct {
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

// GenerateNodeConfig creates the node-config.json content.
func GenerateNodeConfig(config *DeployConfig, result *DeployResult) (*NitroNodeConfig, error) {
	externalSigner := ExternalSignerConfig{
		URL:              "${POPSIGNER_MTLS_URL}",
		Method:           "eth_signTransaction",
		ClientCert:       "/certs/client.crt",
		ClientPrivateKey: "/certs/client.key",
	}

	nodeCfg := &NitroNodeConfig{
		ParentChain: ParentChainConfig{
			Connection: ConnectionConfig{
				URL: "${L1_RPC_URL}",
			},
		},
		Chain: ChainNodeConfig{
			ID:        uint64(config.ChainID),
			InfoFiles: "/config/chain-info.json",
		},
		HTTP: HTTPConfig{
			Addr:       "0.0.0.0",
			Port:       8547,
			VHosts:     "*",
			Corsdomain: "*",
			API:        []string{"eth", "net", "web3", "arb", "debug"},
		},
		WS: WSConfig{
			Addr: "0.0.0.0",
			Port: 8548,
			API:  []string{"eth", "net", "web3"},
		},
		Node: NodeSettings{
			Sequencer: SequencerConfig{Enable: true},
			BatchPoster: BatchPosterConfig{
				Enable:     true,
				DataPoster: DataPosterConfig{ExternalSigner: externalSigner},
			},
			Staker: StakerConfig{
				Enable:     true,
				Strategy:   "MakeNodes",
				DataPoster: DataPosterConfig{ExternalSigner: externalSigner},
			},
			DelayedSequencer: DelayedSeqConfig{Enable: true},
		},
		Metrics: MetricsConfig{
			Server: MetricsServerConfig{
				Addr: "0.0.0.0",
				Port: 9642,
			},
		},
	}

	if config.DataAvailability != "rollup" {
		sequencerInbox := ""
		if result != nil && result.CoreContracts != nil {
			sequencerInbox = result.CoreContracts.SequencerInbox
		}
		nodeCfg.Node.DataAvailability = &DAConfig{
			Enable:             true,
			SequencerInboxAddr: sequencerInbox,
			Celestia: &CelestiaConfig{
				Enable:    true,
				ServerURL: "${CELESTIA_RPC_URL}",
			},
		}
	}

	return nodeCfg, nil
}

// GenerateValidatorNodeConfig creates a node-config.json for a validator-only node.
func GenerateValidatorNodeConfig(config *DeployConfig, result *DeployResult) (*NitroNodeConfig, error) {
	baseCfg, err := GenerateNodeConfig(config, result)
	if err != nil {
		return nil, err
	}

	baseCfg.Node.Sequencer.Enable = false
	baseCfg.Node.BatchPoster.Enable = false
	baseCfg.Execution = &ExecutionConfig{
		ForwardingTarget: "${SEQUENCER_URL}",
	}

	return baseCfg, nil
}
