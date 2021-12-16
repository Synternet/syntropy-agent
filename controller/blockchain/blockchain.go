package blockchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/ipfs"
	"github.com/SyntropyNet/syntropy-agent/pkg/state"
	"github.com/cosmos/go-bip39"
	"github.com/decred/base58"
	"math/rand"
	"os"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/controller"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"time"

	ipfsApi "github.com/ipfs/go-ipfs-api"
)

const pkgName = "Blockchain Controller. "
const mnemonicPath = "/etc/syntropy/mnemonic"
const addressPath = "/etc/syntropy/address"
const reconnectDelay = 10000 // 10 seconds (in milliseconds)
const waitForMsg = time.Duration(1000) * time.Millisecond
const (
	// State machine constants
	stopped = iota
	connecting
	running
)

var ErrNotRunning = errors.New("substrate api is not running")

// Blockchain controller. To be implemented in future
type BlockchainController struct {
	sync.Mutex
	state.StateMachine
	substrateApi *gsrpc.SubstrateAPI
	keyringPair  signature.KeyringPair
	ipfsShell    *ipfsApi.Shell

	url           string
	token         string
	version       string
	address       string
	mnemonic      string
	lastCommodity []byte
}

type BlockchainMsg struct {
	Url string `json:"url"`
	Cid string `json:"cid"`
}

var err = errors.New("blockchain controller not yet implemented")

func New() (controller.Controller, error) {
	url := config.GetCloudURL()

	bc := BlockchainController{
		url:     url,
		token:   config.GetAgentToken(),
		version: config.GetVersion(),
	}
	var (
		mnemonic string
	)
	if _, err := os.Stat(mnemonicPath); err == nil {
		content, err := os.ReadFile(mnemonicPath)
		if err != nil {
			logger.Error().Println(pkgName, err)
		}
		mnemonic = string(content)
	} else if errors.Is(err, os.ErrNotExist) {

		err := os.Mkdir("/etc/syntropy", 0600)
		if err != nil {
			logger.Error().Println(pkgName, err)
		}

		mnemonicFile, err := os.OpenFile(mnemonicPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			logger.Error().Println(pkgName, err)
		}
		entropy, _ := bip39.NewEntropy(256)
		mnemonic, _ := bip39.NewMnemonic(entropy)
		mnemonicFile.WriteString(mnemonic)
		mnemonicFile.Close()

	}
	bc.keyringPair, err = signature.KeyringPairFromSecret(mnemonic, 42)
	if err != nil {
		logger.Error().Println(pkgName, err)
	}

	// Always update address file with latest content.
	addressFile, err := os.OpenFile(addressPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Error().Println(pkgName, err)
	}
	addressFile.WriteString(bc.keyringPair.Address)
	addressFile.Close()

	err = bc.connect()
	if err != nil {
		return nil, err
	}

	return &bc, nil
}

func (bc *BlockchainController) connect() (err error) {
	bc.SetState(connecting)
	for {
		bc.substrateApi, err = gsrpc.NewSubstrateAPI(bc.url)
		logger.Info().Println("CONNECTED TO SUBSTRATE API")
		if err != nil {
			logger.Error().Printf("%s ConnectionError: %s\n", pkgName, err.Error())
			// Add some randomised sleep, so if controller was down
			// the reconnecting agents could DDOS the controller
			delay := time.Duration(rand.Int31n(reconnectDelay)) * time.Millisecond
			logger.Warning().Println(pkgName, "Reconnecting in ", delay)
			time.Sleep(delay)
			continue
		}

		bc.ipfsShell = ipfsApi.NewShell(config.GetIpfsUrl())

		bc.SetState(running)
		break
	}
	return nil
}

func (bc *BlockchainController) Recv() ([]byte, error) {
	if bc.GetState() == stopped {
		return nil, ErrNotRunning
	}
	meta, err := bc.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		logger.Error().Println(pkgName, err)
	}

	for {
		key, err := types.CreateStorageKey(meta, "Commodity", "CommoditiesForAccount", bc.keyringPair.PublicKey, nil)
		if err != nil {
			logger.Error().Println(pkgName, err)
		}
		type Commodity struct {
			ID      types.Hash
			Payload []byte
		}
		var res []Commodity

		_, err = bc.substrateApi.RPC.State.GetStorageLatest(key, &res)
		if err != nil {
			bc.connect()
			continue
		}

		if len(res) == 0 {
			time.Sleep(waitForMsg)
			continue
		}

		if bytes.Equal(bc.lastCommodity, res[len(res)-1].Payload) {
			time.Sleep(waitForMsg)
			continue
		}

		bc.lastCommodity = res[len(res)-1].Payload

		msg := &BlockchainMsg{}
		json.Unmarshal(bc.lastCommodity, &msg)

		data, err := ipfs.GetIpfsPayload(msg.Url)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s", err)
			os.Exit(1)
		}

		return data, nil

	}
}

func (bc *BlockchainController) Write(b []byte) (n int, err error) {

	if controllerState := bc.GetState(); controllerState != running {
		logger.Warning().Println(pkgName, "Controller is not running. Current state: ", controllerState)
		return 0, ErrNotRunning
	}

	bc.Lock()
	defer bc.Unlock()

	reader := bytes.NewReader(b)

	cid, err := bc.ipfsShell.Add(reader)
	ipfsUrl := "https://ipfs.io/ipfs/" + cid

	msg, _ := json.Marshal(BlockchainMsg{
		Url: ipfsUrl,
		Cid: cid,
	})

	meta, err := bc.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}

	genesisHash, err := bc.substrateApi.RPC.Chain.GetBlockHash(0)
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", bc.keyringPair.PublicKey, nil)
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}
	var accountInfo types.AccountInfo
	ok, err := bc.substrateApi.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		logger.Error().Println(pkgName, "Send error: ", err)
	}

	nonce := uint32(accountInfo.Nonce)

	rv, err := bc.substrateApi.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}

	type CommodityInfo struct {
		Info []byte
	}
	c, err := types.NewCall(meta, "Commodity.mint", types.NewAccountID(base58.Decode(config.GetOwnerAddress())), msg)
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}

	ext := types.NewExtrinsic(c)
	err = ext.Sign(bc.keyringPair, types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		GenesisHash:        genesisHash,
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	})
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}

	sub, err := bc.substrateApi.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	}
	defer sub.Unsubscribe()
	sub.Chan()

	n = len(b)
	return n, err
}

func (bc *BlockchainController) Close() error {
	if bc.GetState() == stopped {
		// cannot close already closed connection
		return ErrNotRunning
	}
	bc.SetState(stopped)
	// Maybe notify blockchain about closing?
	return nil
}
