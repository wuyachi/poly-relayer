/*
 * Copyright (C) 2021 The poly network Authors
 * This file is part of The poly network library.
 *
 * The  poly network  is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The  poly network  is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 * You should have received a copy of the GNU Lesser General Public License
 * along with The poly network .  If not, see <http://www.gnu.org/licenses/>.
 */

package zion

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"

	ccm "github.com/devfans/zion-sdk/contracts/native/go_abi/cross_chain_manager_abi"

	"github.com/polynetwork/bridge-common/base"
	"github.com/polynetwork/bridge-common/chains/eth"
	"github.com/polynetwork/bridge-common/chains/zion"
	"github.com/polynetwork/bridge-common/log"
	"github.com/polynetwork/bridge-common/wallet"

	"github.com/polynetwork/poly-relayer/bus"
	"github.com/polynetwork/poly-relayer/config"
	"github.com/polynetwork/poly-relayer/msg"
)

type Submitter struct {
	context.Context
	wg       *sync.WaitGroup
	config   *config.SubmitterConfig
	sdk      *zion.SDK
	name     string
	sync     *config.HeaderSyncConfig
	composer msg.SrcComposer
	state    bus.ChainStore // Header sync marking
	wallet   wallet.IWallet
	signer   *accounts.Account

	// Check last header commit
	lastCommit   uint64
	lastCheck    uint64
	blocksToWait uint64
	txabi        abi.ABI
}

type Composer struct {
	compose msg.PolyComposer
}

func (c *Composer) LatestHeight() (uint64, error) { return 0, nil }
func (c *Composer) Compose(tx *msg.Tx) error      { return c.compose(tx) }

func (s *Submitter) Init(config *config.SubmitterConfig) (err error) {
	s.config = config
	s.name = base.GetChainName(config.ChainId)
	s.blocksToWait = base.BlocksToWait(config.ChainId)
	log.Info("Chain blocks to wait", "blocks", s.blocksToWait, "chain", s.name)
	s.sdk, err = zion.WithOptions(base.POLY, config.Nodes, time.Minute, 1)
	if err != nil {
		return
	}
	if config.Wallet != nil {
		sdk, err := eth.WithOptions(base.POLY, config.Wallet.Nodes, time.Minute, 1)
		if err != nil {
			return err
		}
		s.wallet = wallet.New(config.Wallet, sdk)
		err = s.wallet.Init()
		if err != nil {
			return err
		}
		accounts := s.wallet.Accounts()
		if len(accounts) > 0 {
			s.signer = &accounts[0]
		}
	}

	/*
	s.hsabi, err = abi.JSON(strings.NewReader(hs.HeaderSyncABI))
	if err != nil {
		return
	}
	 */
	s.txabi, err = abi.JSON(strings.NewReader(ccm.CrossChainManagerABI))
	return
}

func (s *Submitter) SDK() *zion.SDK {
	return s.sdk
}

func (s *Submitter) Submit(msg msg.Message) error {
	return nil
}

func (s *Submitter) Hook(ctx context.Context, wg *sync.WaitGroup, ch <-chan msg.Message) error {
	s.Context = ctx
	s.wg = wg
	return nil
}

func (s *Submitter) SubmitHeadersWithLoop(chainId uint64, headers [][]byte, header *msg.Header) (err error) {
	start := time.Now()
	h := uint64(0)
	if len(headers) > 0 {
		err = s.submitHeadersWithLoop(chainId, headers, header)
		if err == nil && header != nil {
			// Check last commit every 4 successful submit
			if s.lastCommit > 0 && s.lastCheck > 3 {
				s.lastCheck = 0
				switch chainId {
				case base.ETH, base.HECO, base.BSC, base.MATIC, base.O3, base.STARCOIN, base.BYTOM, base.HSC:
					height, e := s.GetSideChainHeight(chainId)
					if e != nil {
						log.Error("Get side chain header height failure", "err", e)
					} else if height < s.lastCommit {
						log.Error("Chain header submit confirm check failure", "chain", s.name, "height", height, "last_submit", s.lastCommit)
						err = msg.ERR_HEADER_MISSING
					} else {
						log.Info("Chain header submit confirm check success", "chain", s.name, "height", height, "last_submit", s.lastCommit)
					}
				}
			} else {
				s.lastCheck++
			}
		}
	}
	if header != nil {
		h = header.Height
		if err == nil {
			s.state.HeightMark(h)        // Mark header sync height
			s.lastCommit = header.Height // Mark last commit
		}
	}
	log.Info("Submit headers to poly", "chain", chainId, "size", len(headers), "height", h, "elapse", time.Since(start), "err", err)
	return
}

func (s *Submitter) submitHeadersWithLoop(chainId uint64, headers [][]byte, header *msg.Header) error {
	/*
	attempt := 0
	var ok bool
	for {
		{
			attempt += 1
			_, err = s.SubmitHeaders(chainId, headers)
			if err == nil {
				return nil
			}
			info := err.Error()
			if strings.Contains(info, "parent header not exist") ||
				strings.Contains(info, "missing required field") ||
				strings.Contains(info, "parent block failed") ||
				strings.Contains(info, "span not correct") ||
				strings.Contains(info, "VerifySpan err") {
				//NOTE: reset header height back here
				log.Error("Possible hard fork, will rollback some blocks", "chain", chainId, "err", err)
				return msg.ERR_HEADER_INCONSISTENT
			}
			log.Error("Failed to submit header to poly", "chain", chainId, "err", err)
		}
		select {
		case <-s.Done():
			log.Warn("Header submitter exiting with headers not submitted", "chain", chainId)
			return nil
		default:
			if attempt > 30 || (attempt > 3 && chainId == base.HARMONY) {
				log.Error("Header submit too many failed attempts", "chain", chainId, "attempts", attempt)
				return msg.ERR_HEADER_SUBMIT_FAILURE
			}
			time.Sleep(time.Second)
		}
	}
	 */
	return nil
}

func (s *Submitter) SubmitHeaders(chainId uint64, headers [][]byte) (hash string, err error) {
	/*
	data, err := s.hsabi.Pack("syncBlockHeader", chainId, s.signer.Address, headers)
	if err != nil {
		return
	}

	hash, err = s.wallet.SendWithAccount(*s.signer, utils.HeaderSyncContractAddress, big.NewInt(0), 0, nil, nil, data)
	if err != nil && !strings.Contains(err.Error(), "already known") {
		return
	}
	var height uint64
	var pending bool
	for {
		height, _, pending, err = s.sdk.Node().Confirm(msg.Hash(hash), 0, 100)
		if height > 0 {
			log.Info("Submitted header to poly", "chain", chainId, "hash", hash, "height", height)
			return
		}
		if err == nil && !pending {
			err = fmt.Errorf("Failed to find the transaction %v", err)
			return
		}
		log.Warn("Tx wait confirm timeout", "chain", chainId, "hash", hash, "pending", pending)
	}
	 */
	return
}

func (s *Submitter) submit(tx *msg.Tx) error {
	err := s.composer.Compose(tx)
	if err != nil {
		if strings.Contains(err.Error(), "missing trie node") {
			return msg.ERR_PROOF_UNAVAILABLE
		}
		return err
	}
	if tx.Param == nil || tx.SrcChainId == 0 {
		return fmt.Errorf("%s submitter src tx %s param is missing or src chain id not specified", s.name, tx.SrcHash)
	}

	if !config.CONFIG.AllowMethod(tx.Param.Method) {
		log.Error("Invalid src tx method", "src_hash", tx.SrcHash, "chain", s.name, "method", tx.Param.Method)
		return nil
	}

	if tx.SrcStateRoot == nil {
		tx.SrcStateRoot = []byte{}
	}

	signer := s.signer
	if tx.PolySender != nil {
		signer = tx.PolySender.(*accounts.Account)
	}
	switch tx.SrcChainId {
	case base.NEO, base.ONT:
		if len(tx.SrcStateRoot) == 0 || len(tx.SrcProof) == 0 {
			return fmt.Errorf("%s submitter src tx src state root(%x) or src proof(%x) missing for chain %s with tx %s", s.name, tx.SrcStateRoot, tx.SrcProof, tx.SrcChainId, tx.SrcHash)
		}
	default:
		// For other chains, reversed?
		// Check done tx existence
		done, err := s.sdk.Node().CheckDone(nil, tx.SrcChainId, tx.Param.CrossChainID)
		if err != nil { return err }
		if done {
			log.Info("Tx already imported", "src_hash", tx.SrcHash)
			return nil
		}
	}
	data, err := s.txabi.Pack("importOuterTransfer",
		tx.SrcChainId, uint32(tx.SrcProofHeight),
		tx.SrcProof,
		signer.Address[:],
		tx.SrcEvent,
		tx.SrcStateRoot,
	)
	if err != nil {
		return fmt.Errorf("Pack zion tx failed", "err", err)
	}
	hash, err := s.wallet.SendWithAccount(*signer, zion.CCM_ADDRESS, big.NewInt(0), 0, nil, nil, data)
	/*
		t, err := s.sdk.Node().Native.Ccm.ImportOuterTransfer(
			tx.SrcChainId,
			tx.SrcEvent,
			uint32(tx.SrcProofHeight),
			tx.SrcProof,
			account,
			tx.SrcStateRoot,
			s.signer,
		)
	*/
	if err != nil {
		if strings.Contains(err.Error(), "tx already done") {
			log.Info("Tx already imported", "src_hash", tx.SrcHash, "chain", tx.SrcChainId)
			return nil
		} else if strings.Contains(err.Error(), "already known") {
			return msg.ERR_TX_PENDING
		}
		return fmt.Errorf("Failed to import tx to poly, %v tx src hash %s", err, tx.SrcHash)
	}
	tx.PolyHash = msg.Hash(hash)
	return nil
}

func (s *Submitter) ProcessTx(m *msg.Tx, composer msg.PolyComposer) (err error) {
	if m.Type() != msg.SRC {
		return fmt.Errorf("%s desired message is not poly tx %v", m.Type())
	}
	s.composer = &Composer{composer}
	return s.submit(m)
}

func (s *Submitter) Process(msg msg.Message, composer msg.PolyComposer) error {
	return nil
}

func (s *Submitter) Stop() error {
	s.wg.Wait()
	return nil
}

func (s *Submitter) ReadyBlock() (height uint64) {
	var err error
	switch s.config.ChainId {
	case base.ETH, base.BSC, base.HECO, base.O3, base.MATIC, base.STARCOIN, base.BYTOM, base.HSC:
		var h uint32
		h, err = s.sdk.Node().GetInfoHeight(nil, s.config.ChainId)
		height = uint64(h)
	default:
		height, err = s.composer.LatestHeight()
	}
	if err != nil {
		log.Error("Failed to get ready block height", "chain", s.name, "err", err)
	}
	return
}

func (s *Submitter) consume(account accounts.Account, mq bus.SortedTxBus) error {
	s.wg.Add(1)
	defer s.wg.Done()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	height := s.ReadyBlock()
	for {
		select {
		case <-s.Done():
			log.Info("Submitter is exiting now", "chain", s.name)
			return nil
		default:
		}

		select {
		case <-ticker.C:
			h := s.ReadyBlock()
			if h > 0 && height != h {
				height = h
				log.Info("Current ready block height", "chain", s.name, "height", height)
			}
		default:
		}

		tx, block, err := mq.Pop(s.Context)
		if err != nil {
			log.Error("Bus pop error", "err", err)
			continue
		}
		if tx == nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if block <= height {
			tx.PolySender = &account
			log.Info("Processing src tx", "src_hash", tx.SrcHash, "src_chain", tx.SrcChainId, "dst_chain", tx.DstChainId)
			err = s.submit(tx)
			if err == nil {
				log.Info("Submitted src tx to poly", "src_hash", tx.SrcHash, "poly_hash", tx.PolyHash)
				continue
			}
			block += 1
			if err == msg.ERR_TX_PENDING {
				block += 69
			}
			tx.Attempts++
			log.Error("Submit src tx to poly error", "chain", s.name, "err", err, "proof_height", tx.SrcProofHeight, "next_try", block)
			bus.SafeCall(s.Context, tx, "push back to tx bus", func() error { return mq.Push(context.Background(), tx, block) })
		} else {
			bus.SafeCall(s.Context, tx, "push back to tx bus", func() error { return mq.Push(context.Background(), tx, block) })
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (s *Submitter) run(mq bus.TxBus) error {
	s.wg.Add(1)
	defer s.wg.Done()
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()

	height := s.ReadyBlock()
	refresh := true

	for {
		select {
		case <-s.Done():
			log.Info("Submitter is exiting now", "chain", s.name)
			return nil
		default:
		}

		if refresh {
			select {
			case <-ticker.C:
				refresh = false
				height = s.ReadyBlock()
			default:
			}
		}

		tx, err := mq.Pop(s.Context)
		if err != nil {
			log.Error("Bus pop error", "err", err)
			continue
		}
		if tx == nil {
			time.Sleep(time.Second)
			continue
		}

		log.Debug("Poly submitter checking on src tx", "src_hash", tx.SrcHash, "src_chain", tx.SrcChainId)
		retry := true

		if height == 0 || tx.SrcHeight <= height {
			log.Info("Processing src tx", "src_hash", tx.SrcHash, "src_chain", tx.SrcChainId, "dst_chain", tx.DstChainId)
			err = s.submit(tx)
			if err != nil {
				log.Error("Submit src tx to poly error", "chain", s.name, "err", err, "proof_height", tx.SrcProofHeight)
				tx.Attempts++
			} else {
				log.Info("Submitted src tx to poly", "src_hash", tx.SrcHash, "poly_hash", tx.PolyHash)
				retry = false
			}
			if height == 0 {
				refresh = true
			}
		} else {
			refresh = true
		}

		if retry {
			bus.SafeCall(s.Context, tx, "push back to tx bus", func() error { return mq.Push(context.Background(), tx) })
		}
	}
}

func (s *Submitter) Start(ctx context.Context, wg *sync.WaitGroup, bus bus.TxBus, delay bus.DelayedTxBus, compose msg.PolyComposer) error {
	return nil
}

func (s *Submitter) Run(ctx context.Context, wg *sync.WaitGroup, mq bus.SortedTxBus, composer msg.SrcComposer) error {
	s.composer = composer
	s.Context = ctx
	s.wg = wg

	accounts := s.wallet.Accounts()
	if len(accounts) == 0 {
		log.Warn("No account available for submitter workers", "chain", s.name)
	}
	for i, a := range accounts {
		log.Info("Starting zion submitter worker", "index", i, "total", len(accounts), "account", a.Address, "chain", s.name, "topic", mq.Topic())
		go s.consume(a, mq)
	}
	return nil
}

func (s *Submitter) StartSync(
	ctx context.Context, wg *sync.WaitGroup, config *config.HeaderSyncConfig,
	reset chan<- uint64, state bus.ChainStore,
) (ch chan msg.Header, err error) {
	s.Context = ctx
	s.wg = wg
	s.sync = config
	s.state = state

	if s.sync.Batch == 0 {
		s.sync.Batch = 1
	}
	if s.sync.Buffer == 0 {
		s.sync.Buffer = 2 * s.sync.Batch
	}
	if s.sync.Timeout == 0 {
		s.sync.Timeout = 1
	}

	if s.sync.ChainId == 0 {
		return nil, fmt.Errorf("Invalid header sync side chain id")
	}

	ch = make(chan msg.Header, s.sync.Buffer)
	go s.startSync(ch, reset)
	return
}

func (s *Submitter) GetSideChainHeight(chainId uint64) (height uint64, err error) {
	h, err := s.sdk.Node().GetInfoHeight(nil, chainId)
	height = uint64(h)
	return
}

func (s *Submitter) syncHeaderLoop(ch <-chan msg.Header, reset chan<- uint64) {
	for {
		select {
		case <-s.Done():
			return
		case header, ok := <-ch:
			if !ok {
				return
			}
			// NOTE err reponse here will revert header sync with delta - 2
			headers := [][]byte{header.Data}
			if header.Data == nil {
				headers = nil
			}
			err := s.SubmitHeadersWithLoop(s.sync.ChainId, headers, &header)
			if err != nil {
				reset <- header.Height - 2
			}
		}
	}
}

func (s *Submitter) syncHeaderBatchLoop(ch <-chan msg.Header, reset chan<- uint64) {
	headers := [][]byte{}
	commit := false
	duration := time.Duration(s.sync.Timeout) * time.Second
	var (
		height uint64
		hdr    *msg.Header
	)

COMMIT:
	for {
		select {
		case <-s.Done():
			break COMMIT
		case header, ok := <-ch:
			if ok {
				hdr = &header
				if len(headers) > 0 && height != header.Height-1 {
					log.Info("Resetting header set", "chain", s.sync.ChainId, "height", height, "current_height", header.Height)
					headers = [][]byte{}
				}
				height = header.Height
				if hdr.Data == nil {
					// Update header sync height
					commit = true
				} else {
					headers = append(headers, header.Data)
					commit = len(headers) >= s.sync.Batch
				}
			} else {
				commit = len(headers) > 0
				break COMMIT
			}
		case <-time.After(duration):
			commit = len(headers) > 0
		}
		if commit {
			commit = false
			// NOTE err reponse here will revert header sync with delta -100
			err := s.SubmitHeadersWithLoop(s.sync.ChainId, headers, hdr)
			if err != nil {
				reset <- height - uint64(len(headers)) - 2
			}
			headers = [][]byte{}
		}
	}
	if len(headers) > 0 {
		s.SubmitHeadersWithLoop(s.sync.ChainId, headers, hdr)
	}
}

func (s *Submitter) startSync(ch <-chan msg.Header, reset chan<- uint64) {
	if s.sync.Batch == 1 {
		s.syncHeaderLoop(ch, reset)
	} else {
		s.syncHeaderBatchLoop(ch, reset)
	}
	log.Info("Header sync exiting loop now")
}

func (s *Submitter) Poly() *zion.SDK {
	return s.sdk
}

func (s *Submitter) ProcessEpochs(epochs []*msg.Tx) (err error) {
	if len(epochs) == 0 {
		return
	}

	headers := [][]byte{}
	for _, m := range epochs {
		if m.Type() != msg.POLY_EPOCH || m.PolyEpoch == nil {
			err = fmt.Errorf("Invalid side chainy epoch message %s", m.Encode())
			return
		}
		headers = append(headers, m.PolyEpoch.Header)
	}

	epoch := epochs[len(epochs)-1].PolyEpoch
	log.Info("Submitting side chain epoch", "epoch", epoch.EpochId, "height", epoch.Height, "chain", s.name, "size", len(epochs), "from_chain", epoch.ChainId)
	hash, err := s.SubmitHeaders(epoch.ChainId, headers)
	log.Info("Submit side chain epochs to zion", "size", len(epochs), "epoch", epoch.EpochId, "height", epoch.Height, "chain", s.name, "from_chain", epoch.ChainId, "hash", hash, "err", err)
	return
}