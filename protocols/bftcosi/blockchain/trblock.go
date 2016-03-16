package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"

	"github.com/dedis/cothority/lib/hashid"
	"github.com/dedis/cothority/lib/proof"
)

type Block struct {
	Magic      [4]byte
	BlockSize  uint32
	HeaderHash string
	*Header
	TransactionList
}

type TrBlock struct {
	Block
}

func (tr *TrBlock) MarshalBinary() ([]byte, error) {
	return json.Marshal(tr)
}

// Hash returns a hash representation of the block
func (tr *TrBlock) HashSum() []byte {
	h := sha256.New()
	h.Write(tr.Magic[:])
	binary.Write(h, binary.LittleEndian, tr.BlockSize)
	h.Write([]byte(tr.HeaderHash))
	h.Write(tr.Header.HashSum())
	h.Write(tr.TransactionList.HashSum())
	return h.Sum(nil)
}

type Header struct {
	MerkleRoot string
	Parent     string
	ParentKey  string
	PublicKey  string
	LeaderId   net.IP
}

// Hash returns a hash representation of the header
func (h *Header) HashSum() []byte {
	ha := sha256.New()
	ha.Write([]byte(h.MerkleRoot))
	ha.Write([]byte(h.Parent))
	ha.Write([]byte(h.ParentKey))
	ha.Write([]byte(h.PublicKey))
	return ha.Sum(nil)
}

func (trb *TrBlock) NewTrBlock(transactions TransactionList, header *Header) *TrBlock {
	return NewTrBlock(transactions, header)
}

func (t *TrBlock) NewHeader(transactions TransactionList, parent string, parentKey string) *Header {
	return NewHeader(transactions, parent, parentKey)
}

func (trb *Block) Calculate_root(transactions TransactionList) (res string) {
	return HashRootTransactions(transactions)
}

// Porting to public method non related to Header / TrBlock whatsoever
func NewTrBlock(transactions TransactionList, header *Header) *TrBlock {
	trb := new(TrBlock)
	trb.Magic = [4]byte{0xF9, 0xBE, 0xB4, 0xD9}
	trb.HeaderHash = trb.Hash(header)
	trb.TransactionList = transactions
	trb.BlockSize = 0
	trb.Header = header
	return trb
}

func NewHeader(transactions TransactionList, parent, parentKey string) *Header {
	hdr := new(Header)
	hdr.Parent = parent
	hdr.ParentKey = parentKey
	hdr.MerkleRoot = HashRootTransactions(transactions)
	return hdr
}
func HashRootTransactions(transactions TransactionList) string {
	var hashes []hashid.HashId

	for _, t := range transactions.Txs {
		temp, _ := hex.DecodeString(t.Hash)
		hashes = append(hashes, temp)
	}
	out, _ := proof.ProofTree(sha256.New, hashes)
	return hex.EncodeToString(out)
}

func (trb *Block) Hash(h *Header) (res string) {
	//change it to be more portable
	return HashHeader(h)

}

func HashHeader(h *Header) string {
	data := fmt.Sprintf("%v", h)
	sha := sha256.New()
	sha.Write([]byte(data))
	hash := sha.Sum(nil)
	return hex.EncodeToString(hash)
}
