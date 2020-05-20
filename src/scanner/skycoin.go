// Package scanner scans bitcoin blockchain and check all transactions
// to see if there are addresses in vout that can match our deposit addresses.
// If found, then generate an event and push to deposit event channel
//
// current scanner doesn't support reconnect after btcd shutdown, if
// any error occur when call btcd apis, the scan service will be closed.
package scanner

import (
	"time"

	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/MDLlife/MDL/src/api"
	"github.com/MDLlife/MDL/src/readable"
)

// SKYScanner blockchain scanner to check if there're deposit coins
type SKYScanner struct {
	log          logrus.FieldLogger
	Base         CommonScanner
	skyRPCClient SkyRPCClient
}

// NewSkycoinScanner creates scanner instance
func NewSkycoinScanner(log logrus.FieldLogger, store Storer, client SkyRPCClient, cfg Config) (*SKYScanner, error) {
	bs := NewBaseScanner(store, log.WithField("prefix", "scanner.sky"), CoinTypeSKY, cfg)

	return &SKYScanner{
		skyRPCClient: client,
		log:          log.WithField("prefix", "scanner.sky"),
		Base:         bs,
	}, nil
}

// Run starts the scanner
func (s *SKYScanner) Run() error {
	return s.Base.Run(s.GetBlockCount, s.getBlockAtHeight, s.waitForNextBlock, s.scanBlock)
}

// Shutdown shutdown the scanner
func (s *SKYScanner) Shutdown() {
	s.log.Info("Closing SKY scanner")
	s.skyRPCClient.Shutdown()
	s.Base.Shutdown()
	s.log.Info("Waiting for SKY scanner to stop")
	s.log.Info("SKY scanner stopped")
}

// scanBlock scans for a new SKY block every ScanPeriod.
// When a new block is found, it compares the block against our scanning
// deposit addresses. If a matching deposit is found, it saves it to the DB.
func (s *SKYScanner) scanBlock(block *CommonBlock) (int, error) {
	log := s.log.WithField("hash", block.Hash)
	log = log.WithField("height", block.Height)

	log.Debug("Scanning block")

	dvs, err := s.Base.GetStorer().ScanBlock(block, CoinTypeSKY)
	if err != nil {
		log.WithError(err).Error("store.ScanBlock failed")
		return 0, err
	}

	log = log.WithField("scannedDeposits", len(dvs))
	log.Infof("Counted %d deposits from block", len(dvs))

	n := 0
	for _, dv := range dvs {
		select {
		case s.Base.GetScannedDepositChan() <- dv:
			n++
		case <-s.Base.GetQuitChan():
			return n, errQuit
		}
	}

	return n, nil
}

// skyBlock2CommonBlock convert skycoin block to common block
func skyBlock2CommonBlock(block *readable.Block) (*CommonBlock, error) {
	if block == nil {
		return nil, ErrEmptyBlock
	}
	cb := CommonBlock{}
	cb.Hash = block.Head.Hash
	cb.Height = int64(block.Head.BkSeq)
	cb.RawTx = make([]CommonTx, 0, len(block.Body.Transactions))
	for _, tx := range block.Body.Transactions {
		cbTx := CommonTx{}
		cbTx.Txid = tx.Hash
		cbTx.Vout = make([]CommonVout, 0, len(tx.Out))
		for i, v := range tx.Out {
			amt, ee := strconv.ParseFloat(v.Coins, 16)
			if ee != nil {
				continue
			}
			cv := CommonVout{}
			cv.N = uint32(i)
			cv.Value = int64(amt * 1e6)
			cv.Addresses = []string{v.Address}
			cbTx.Vout = append(cbTx.Vout, cv)
		}
		cb.RawTx = append(cb.RawTx, cbTx)
	}

	return &cb, nil
}

// GetBlockCount returns the hash and height of the block in the longest (best) chain.
func (s *SKYScanner) GetBlockCount() (int64, error) {
	rb, err := s.skyRPCClient.GetLastBlocks()
	if err != nil {
		return 0, err
	}

	return int64(rb.Head.BkSeq), nil
}

// getBlock returns block of given hash
func (s *SKYScanner) getBlock(seq int64) (*CommonBlock, error) {
	rb, err := s.skyRPCClient.GetBlocksBySeq(uint64(seq))
	if err != nil {
		return nil, err
	}

	return skyBlock2CommonBlock(rb)
}

// getBlockAtHeight returns that block at a specific height
func (s *SKYScanner) getBlockAtHeight(seq int64) (*CommonBlock, error) {
	b, err := s.getBlock(seq)
	return b, err
}

// getNextBlock returns the next block of given hash, return nil if next block does not exist
func (s *SKYScanner) getNextBlock(seq uint64) (*CommonBlock, error) {
	b, err := s.skyRPCClient.GetBlocksBySeq(seq + 1)
	if err != nil {
		return nil, err
	}
	return skyBlock2CommonBlock(b)
}

// waitForNextBlock scans for the next block until it is available
func (s *SKYScanner) waitForNextBlock(block *CommonBlock) (*CommonBlock, error) {
	log := s.log.WithField("blockHash", block.Hash)
	log = log.WithField("blockHeight", block.Height)
	log.Debug("Waiting for the next block")

	for {
		nextBlock, err := s.getNextBlock(uint64(block.Height))
		if err != nil {
			//if err == ErrEmptyBlock {
			log.WithError(err).Debug("getNextBlock empty")
			//} else {
			//	log.WithError(err).Error("getNextBlock failed")
			//}
		}
		if nextBlock == nil {
			log.Debug("No new block yet")
		}
		if err != nil || nextBlock == nil {
			select {
			case <-s.Base.GetQuitChan():
				return nil, errQuit
			case <-time.After(s.Base.GetScanPeriod()):
				continue
			}
		}

		log.WithFields(logrus.Fields{
			"hash":   nextBlock.Hash,
			"height": nextBlock.Height,
		}).Debug("Found nextBlock")

		return nextBlock, nil
	}
}

// AddScanAddress adds new scan address
func (s *SKYScanner) AddScanAddress(addr, coinType string) error {
	return s.Base.GetStorer().AddScanAddress(addr, coinType)
}

// GetScanAddresses returns the deposit addresses that need to scan
func (s *SKYScanner) GetScanAddresses() ([]string, error) {
	return s.Base.GetStorer().GetScanAddresses(CoinTypeSKY)
}

// GetDeposit returns deposit value channel.
func (s *SKYScanner) GetDeposit() <-chan DepositNote {
	return s.Base.GetDeposit()
}

// SkyClient provides methods for sending coins
type SkyClient struct {
	//walletFile   string
	//changeAddr   string
	skyRPCClient *api.Client
}

// NewSkyClient creates RPC instance
func NewSkyClient(server, port string) *SkyClient {
	rpcClient := api.NewClient("http://" + server + ":" + port)

	return &SkyClient{
		skyRPCClient: rpcClient,
	}
}

// GetTransaction returns transaction by txid
func (c *SkyClient) GetTransaction(txid string) (*readable.TransactionWithStatus, error) {
	return c.skyRPCClient.Transaction(txid)
}

// GetBlocks get blocks from RPC
func (c *SkyClient) GetBlocks(start, end uint64) (*readable.Blocks, error) {
	return c.skyRPCClient.BlocksInRange(start, end)
}

// GetBlocksBySeq get blocks by seq
func (c *SkyClient) GetBlocksBySeq(seq uint64) (*readable.Block, error) {
	return c.skyRPCClient.BlockBySeq(seq)
}

// GetLastBlocks get last blocks
func (c *SkyClient) GetLastBlocks() (*readable.Block, error) {
	var blocks *readable.Blocks
	blocks, err := c.skyRPCClient.LastBlocks(1)
	if err != nil {
		return nil, err
	}

	if len(blocks.Blocks) == 0 {
		return nil, nil
	}
	return &blocks.Blocks[0], nil
}

// Shutdown the node
func (c *SkyClient) Shutdown() {
}
