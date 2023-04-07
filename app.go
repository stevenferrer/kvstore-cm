package kvstore

import (
	"bytes"
	"log"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/dgraph-io/badger/v3"
)

type App struct {
	db           *badger.DB
	onGoingBlock *badger.Txn
}

var _ abcitypes.Application = (*App)(nil)

func NewApp(db *badger.DB) *App {
	return &App{db: db}
}

func (a *App) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	var (
		code  = uint32(0)
		valid = isValid(req.Tx)
	)

	// non-zero code is considered invalid
	if !valid {
		code = 1
	}

	return abcitypes.ResponseCheckTx{Code: code}
}

func (a *App) BeginBlock(block abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	a.onGoingBlock = a.db.NewTransaction(true)
	return abcitypes.ResponseBeginBlock{}
}

func (a *App) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	if valid := isValid(req.Tx); !valid {
		return abcitypes.ResponseDeliverTx{Code: 1}
	}

	parts := bytes.SplitN(req.Tx, []byte("="), 2)
	key, value := parts[0], parts[1]

	if err := a.onGoingBlock.Set(key, value); err != nil {
		log.Panicf("error writing to database, unable to execute tx: %v", err)
	}

	return abcitypes.ResponseDeliverTx{Code: 0}
}

func (a *App) EndBlock(block abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	log.Println("endblock")
	return abcitypes.ResponseEndBlock{}
}

func (a *App) Commit() abcitypes.ResponseCommit {
	if err := a.onGoingBlock.Commit(); err != nil {
		log.Panicf("error writing to database, unable to commit block: %v", err)
	}
	return abcitypes.ResponseCommit{Data: []byte{}}
}

func (a *App) Info(info abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

func (a *App) Query(req abcitypes.RequestQuery) abcitypes.ResponseQuery {
	resp := abcitypes.ResponseQuery{Key: req.Data}

	err := a.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(req.Data)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
			resp.Log = "key does not exist"
			return nil
		}

		return item.Value(func(val []byte) error {
			resp.Log = "exists"
			resp.Value = val
			return nil
		})
	})
	if err != nil {
		log.Panicf("error reading database, unable to execute query: %v", err)
	}

	return resp
}

func (a *App) PrepareProposal(proposal abcitypes.RequestPrepareProposal) abcitypes.ResponsePrepareProposal {
	// application can modify the tx group (reorder, add or remove)
	return abcitypes.ResponsePrepareProposal{Txs: proposal.Txs}
}

func (a *App) ProcessProposal(proposal abcitypes.RequestProcessProposal) abcitypes.ResponseProcessProposal {
	// can accept or reject proposal
	return abcitypes.ResponseProcessProposal{Status: abcitypes.ResponseProcessProposal_ACCEPT}
}

func (a *App) InitChain(chain abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

func (a *App) ListSnapshots(snapshots abcitypes.RequestListSnapshots) abcitypes.ResponseListSnapshots {
	return abcitypes.ResponseListSnapshots{}
}

func (a *App) OfferSnapshot(snapshot abcitypes.RequestOfferSnapshot) abcitypes.ResponseOfferSnapshot {
	return abcitypes.ResponseOfferSnapshot{}
}

func (a *App) LoadSnapshotChunk(chunk abcitypes.RequestLoadSnapshotChunk) abcitypes.ResponseLoadSnapshotChunk {
	return abcitypes.ResponseLoadSnapshotChunk{}
}

func (a *App) ApplySnapshotChunk(chunk abcitypes.RequestApplySnapshotChunk) abcitypes.ResponseApplySnapshotChunk {
	return abcitypes.ResponseApplySnapshotChunk{}
}

func isValid(tx []byte) bool {
	// check tx format
	parts := bytes.Split(tx, []byte("="))
	return len(parts) == 2
}
