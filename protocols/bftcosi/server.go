package byzcoin

import (
	"sync"

	"github.com/dedis/cothority/lib/dbg"
	"github.com/dedis/cothority/lib/sda"
	"github.com/dedis/cothority/protocols/byzcoin/blockchain/blkparser"
	"github.com/satori/go.uuid"
)

type BlockServer interface {
	AddTransaction(blkparser.Tx)
	Instantiate(n *sda.Node) (sda.ProtocolInstance, error)
}

// ByzCoinServer is the longterm control service that listens for transactions and
// dispatch them to a new ByzCoin for each new signing that we want to do.
// It creates the ByzCoin protocols and run them. only used by the root since
// only the root pariticipates to the creation of the block.
type ByzCoinServer struct {
	// transactions pool where all the incoming transactions are stored
	transactions []blkparser.Tx
	// lock associated
	transactionLock sync.Mutex
	// how many transactions should we give to an instance
	blockSize int
	timeOutMs uint64
	fail      uint
	// all the protocols byzcoin he generated.Map from RoundID <-> ByzCoin
	// protocol instance.
	instances map[uuid.UUID]*ByzCoin
	// blockSignatureChan is the channel used to pass out the signatures that
	// ByzCoin's instances have made
	blockSignatureChan chan BlockSignature
	// enoughBlock signals the server we have enough
	// no comments..
	transactionChan chan blkparser.Tx
	requestChan     chan bool
	responseChan    chan []blkparser.Tx
}

// NewByzCoinServer returns a new fresh ByzCoinServer. It must be given the blockSize in order
// to efficiently give the transactions to the ByzCoin instances.
func NewByzCoinServer(blockSize int, timeOutMs uint64, fail uint) *ByzCoinServer {
	s := &ByzCoinServer{
		blockSize:          blockSize,
		timeOutMs:          timeOutMs,
		fail:               fail,
		instances:          make(map[uuid.UUID]*ByzCoin),
		blockSignatureChan: make(chan BlockSignature),
		transactionChan:    make(chan blkparser.Tx),
		requestChan:        make(chan bool),
		responseChan:       make(chan []blkparser.Tx),
	}
	go s.listenEnoughBlocks()
	return s
}

func (s *ByzCoinServer) AddTransaction(tr blkparser.Tx) {
	s.transactionChan <- tr
}

// ListenClientTransactions will bind to a port a listen for incoming connection
// from clients. These client will be able to pass the transactions to the
// server.
func (s *ByzCoinServer) ListenClientTransactions() {
	panic("not implemented yet")
}

// Instantiate takes blockSize transactions and create the byzcoin instances.
func (s *ByzCoinServer) Instantiate(node *sda.Node) (sda.ProtocolInstance, error) {
	// wait until we have enough blocks
	currTransactions := s.waitEnoughBlocks()
	dbg.Lvl1("Instantiate ByzCoin Round with", len(currTransactions), " transactions")
	pi, err := NewByzCoinRootProtocol(node, currTransactions, s.timeOutMs, s.fail)
	node.SetProtocolInstance(pi)

	return pi, err
}

// BlockSignature returns a channel that is given each new block signature as
// soon as they are arrive (Wether correct or not).
func (s *ByzCoinServer) BlockSignaturesChan() <-chan BlockSignature {
	return s.blockSignatureChan
}

func (s *ByzCoinServer) onDoneSign(blk BlockSignature) {
	s.blockSignatureChan <- blk
}

func (s *ByzCoinServer) waitEnoughBlocks() []blkparser.Tx {
	s.requestChan <- true
	transactions := <-s.responseChan
	return transactions
}

func (s *ByzCoinServer) listenEnoughBlocks() {
	// TODO the server should have a transaction pool instead:
	var transactions []blkparser.Tx
	var want bool
	for {
		select {
		case tr := <-s.transactionChan:
			// FIXME this will lead to a very large slice if the client sends many
			if len(transactions) < s.blockSize {
				transactions = append(transactions, tr)
			}
			if want {
				if len(transactions) >= s.blockSize {
					s.responseChan <- transactions[:s.blockSize]
					transactions = transactions[s.blockSize:]
					want = false
				}
			}
		case <-s.requestChan:
			want = true
			if len(transactions) >= s.blockSize {
				s.responseChan <- transactions[:s.blockSize]
				transactions = transactions[s.blockSize:]
				want = false
			}
		}
	}
}

type NtreeServer struct {
	*ByzCoinServer
}

func NewNtreeServer(blockSize int) *NtreeServer {
	ns := new(NtreeServer)
	// we dont care about timeout + fail in Naive comparison
	ns.ByzCoinServer = NewByzCoinServer(blockSize, 0, 0)
	return ns
}

func (nt *NtreeServer) Instantiate(node *sda.Node) (sda.ProtocolInstance, error) {
	dbg.Lvl2("NtreeServer waiting enough transactions...")
	currTransactions := nt.waitEnoughBlocks()
	pi, err := NewNTreeRootProtocol(node, currTransactions)
	node.SetProtocolInstance(pi)
	dbg.Lvl1("NtreeServer instantiated Ntree Root Protocol with", len(currTransactions), " transactions")
	return pi, err
}
