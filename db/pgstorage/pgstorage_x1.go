package pgstorage

import (
	"context"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// GetDepositsWithLeafType gets the deposit list which be smaller than depositCount.
func (p *PostgresStorage) GetDepositsWithLeafType(ctx context.Context, destAddr string, limit uint, offset uint, leafType uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = `
		SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE dest_addr = $1 AND leaf_type = $4
		ORDER BY d.block_id DESC, d.deposit_cnt DESC LIMIT $2 OFFSET $3`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getDepositsSQL, common.FromHex(destAddr), limit, offset, leafType)
	return p.convertDepositBase(rows, err)
}

func (p *PostgresStorage) convertDepositBase(rows pgx.Rows, err error) ([]*etherman.Deposit, error) {
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.Id, &deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim, &deposit.Time)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}

	return deposits, nil
}

// GetDepositByHash returns a deposit from a specific account and tx hash
func (p *PostgresStorage) GetDepositByHash(ctx context.Context, destAddr string, networkID uint, txHash string, dbTx pgx.Tx) (*etherman.Deposit, error) {
	var (
		deposit etherman.Deposit
		amount  string
	)
	const getDepositSQL = `
		SELECT leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE d.dest_addr = $1 AND d.network_id = $2 AND d.tx_hash = $3`
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getDepositSQL, common.HexToAddress(destAddr), networkID, common.HexToHash(txHash)).Scan(
		&deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
		&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, gerror.ErrStorageNotFound
	}
	deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd

	return &deposit, err
}

// GetPendingTransactions gets all the deposit transactions of a user that have not been claimed
func (p *PostgresStorage) GetPendingTransactions(ctx context.Context, destAddr string, limit uint, offset uint, leafType uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = `SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE dest_addr = $1 AND leaf_type = $4 AND NOT EXISTS
			(SELECT 1 FROM sync.claim as c WHERE c.index = d.deposit_cnt AND c.network_id = d.dest_net)
		ORDER BY d.block_id DESC, d.deposit_cnt DESC LIMIT $2 OFFSET $3`

	return p.getDepositList(ctx, getDepositsSQL, dbTx, common.FromHex(destAddr), limit, offset, leafType)
}

// GetNotReadyTransactions returns all the deposit transactions with ready_for_claim = false
func (p *PostgresStorage) GetNotReadyTransactions(ctx context.Context, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = `SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE ready_for_claim = false
		ORDER BY d.block_id DESC, d.deposit_cnt DESC LIMIT $1 OFFSET $2`

	return p.getDepositList(ctx, getDepositsSQL, dbTx, limit, offset)
}

func (p *PostgresStorage) GetReadyPendingTransactions(ctx context.Context, networkID uint, leafType uint, limit uint, offset uint, minReadyTime time.Time, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = `SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE d.network_id = $1 AND ready_for_claim = true AND leaf_type = $4 AND ready_time >= $5 AND NOT EXISTS
			(SELECT 1 FROM sync.claim as c WHERE c.index = d.deposit_cnt AND c.network_id = d.dest_net)
		ORDER BY d.block_id DESC, d.deposit_cnt DESC LIMIT $2 OFFSET $3`

	return p.getDepositList(ctx, getDepositsSQL, dbTx, networkID, limit, offset, leafType, minReadyTime)
}

func (p *PostgresStorage) getDepositList(ctx context.Context, sql string, dbTx pgx.Tx, args ...interface{}) ([]*etherman.Deposit, error) {
	rows, err := p.getExecQuerier(dbTx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.Id, &deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim, &deposit.Time)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}

	return deposits, nil
}

func (p *PostgresStorage) GetNotReadyTransactionsWithBlockRange(ctx context.Context, networkID uint, minBlockNum, maxBlockNum uint64, limit, offset uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = `SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE b.network_id = $1 AND ready_for_claim = false AND b.block_num >= $2 AND b.block_num <= $3
		ORDER BY d.block_id DESC, d.deposit_cnt DESC LIMIT $4 OFFSET $5`

	rows, err := p.getExecQuerier(dbTx).Query(ctx, getDepositsSQL, networkID, minBlockNum, maxBlockNum, limit, offset)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.Id, &deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim, &deposit.Time)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}

	return deposits, nil
}

// GetL1Deposits get the L1 deposits remain to be ready_for_claim
func (p *PostgresStorage) GetL1Deposits(ctx context.Context, exitRoot []byte, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const updateDepositsStatusSQL = `Select d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
			FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
			WHERE deposit_cnt <= (SELECT deposit_cnt FROM mt.root WHERE root = $1 AND network = 0) 
			AND d.network_id = 0 AND ready_for_claim = false`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, updateDepositsStatusSQL, exitRoot)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))
	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.Id, &deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim, &deposit.Time)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}
	return deposits, nil
}

func (p *PostgresStorage) UpdateL1DepositStatus(ctx context.Context, depositCount uint, eventTime time.Time, dbTx pgx.Tx) error {
	const updateDepositStatusSQL = `UPDATE sync.deposit SET ready_for_claim = true, ready_time = $1 
		WHERE deposit_cnt = $2 And network_id = 0`
	_, err := p.getExecQuerier(dbTx).Exec(ctx, updateDepositStatusSQL, eventTime, depositCount)
	return err
}

// UpdateL2DepositsStatus updates the ready_for_claim status of L2 deposits. and return deposit list
func (p *PostgresStorage) UpdateL2DepositsStatusX1(ctx context.Context, exitRoot []byte, eventTime time.Time, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const updateDepositsStatusSQL = `UPDATE sync.deposit SET ready_for_claim = true, ready_time = $1
		WHERE deposit_cnt <=
			(SELECT deposit_cnt FROM mt.root WHERE root = $2 AND network = 1)
			AND network_id = 1 AND ready_for_claim = false
			RETURNING leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, network_id, tx_hash, metadata, ready_for_claim;`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, updateDepositsStatusSQL, eventTime, exitRoot)
	if err != nil {
		return nil, err
	}
	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))
	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}
	return deposits, nil
}

func (p *PostgresStorage) GetClaimTxsByStatusWithLimit(ctx context.Context, statuses []ctmtypes.MonitoredTxStatus, limit uint, offset uint, dbTx pgx.Tx) ([]ctmtypes.MonitoredTx, error) {
	const getMonitoredTxsSQL = "SELECT deposit_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at FROM sync.monitored_txs WHERE status = ANY($1) ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getMonitoredTxsSQL, pq.Array(statuses), limit, offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return []ctmtypes.MonitoredTx{}, nil
	} else if err != nil {
		return nil, err
	}

	mTxs := make([]ctmtypes.MonitoredTx, 0, len(rows.RawValues()))
	for rows.Next() {
		var (
			value   string
			history [][]byte
		)
		mTx := ctmtypes.MonitoredTx{}
		err = rows.Scan(&mTx.DepositID, &mTx.From, &mTx.To, &mTx.Nonce, &value, &mTx.Data, &mTx.Gas, &mTx.Status, pq.Array(&history), &mTx.CreatedAt, &mTx.UpdatedAt)
		if err != nil {
			return mTxs, err
		}
		mTx.Value, _ = new(big.Int).SetString(value, 10) //nolint:gomnd
		mTx.History = make(map[common.Hash]bool)
		for _, h := range history {
			mTx.History[common.BytesToHash(h)] = true
		}
		mTxs = append(mTxs, mTx)
	}

	return mTxs, nil
}

// GetClaimTxById gets the monitored transactions by id (depositCount)
func (p *PostgresStorage) GetClaimTxById(ctx context.Context, id uint, dbTx pgx.Tx) (*ctmtypes.MonitoredTx, error) {
	getClaimSql := "SELECT deposit_id, from_addr, to_addr, nonce, value, data, gas, status, history, created_at, updated_at FROM sync.monitored_txs WHERE id = $1"
	var (
		value   string
		history [][]byte
		mTx     = &ctmtypes.MonitoredTx{}
	)
	err := p.getExecQuerier(dbTx).QueryRow(ctx, getClaimSql, id).
		Scan(&mTx.DepositID, &mTx.From, &mTx.To, &mTx.Nonce, &value, &mTx.Data, &mTx.Gas, &mTx.Status, pq.Array(&history), &mTx.CreatedAt, &mTx.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gerror.ErrStorageNotFound
		}
		return nil, err
	}
	mTx.Value, _ = new(big.Int).SetString(value, 10) //nolint:gomnd
	mTx.History = make(map[common.Hash]bool)
	for _, h := range history {
		mTx.History[common.BytesToHash(h)] = true
	}

	return mTx, nil
}

// GetAllMainCoins returns all the coin info from the main_coins table
func (p *PostgresStorage) GetAllMainCoins(ctx context.Context, limit uint, offset uint, dbTx pgx.Tx) ([]*pb.CoinInfo, error) {
	const getCoinsSQL = `SELECT symbol, name, decimals, encode(address, 'hex'), chain_id, network_id, logo_link
		FROM common.main_coins WHERE is_deleted = false ORDER BY id LIMIT $1 OFFSET $2`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, getCoinsSQL, limit, offset)
	if err != nil {
		return nil, err
	}

	var result []*pb.CoinInfo
	for rows.Next() {
		coin := &pb.CoinInfo{}
		err = rows.Scan(&coin.Symbol, &coin.Name, &coin.Decimals, &coin.Address, &coin.ChainId, &coin.NetworkId, &coin.LogoLink)
		if err != nil {
			log.Errorf("GetAllMainCoins scan row error[%v]", err)
			return nil, err
		}
		if coin.Address != "" {
			coin.Address = "0x" + coin.Address
		}
		result = append(result, coin)
	}
	return result, nil
}

// GetLatestReadyDeposits returns the latest deposit transactions with ready_for_claim = true
func (p *PostgresStorage) GetLatestReadyDeposits(ctx context.Context, networkID uint, limit uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const getDepositsSQL = `
		SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at, ready_time
		FROM sync.deposit as d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id
		WHERE d.network_id = $1 AND ready_for_claim = true AND ready_time IS NOT NULL
		ORDER BY d.deposit_cnt DESC LIMIT $2`

	rows, err := p.getExecQuerier(dbTx).Query(ctx, getDepositsSQL, networkID, limit)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))

	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.Id, &deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim, &deposit.Time, &deposit.ReadyTime)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}

	return deposits, nil
}

// UpdateL1DepositsStatusX1 updates the ready_for_claim status of L1 deposits.
func (p *PostgresStorage) UpdateL1DepositsStatusX1(ctx context.Context, exitRoot []byte, eventTime time.Time, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	const updateDepositsStatusSQL = `WITH d AS (UPDATE sync.deposit SET ready_for_claim = true, ready_time = $1 
		WHERE deposit_cnt <=
			(SELECT deposit_cnt FROM mt.root WHERE root = $2 AND network = 0) 
			AND network_id = 0 AND ready_for_claim = false
			RETURNING *)
		SELECT d.id, leaf_type, orig_net, orig_addr, amount, dest_net, dest_addr, deposit_cnt, block_id, b.block_num, d.network_id, tx_hash, metadata, ready_for_claim, b.received_at
		FROM d INNER JOIN sync.block as b ON d.network_id = b.network_id AND d.block_id = b.id`
	rows, err := p.getExecQuerier(dbTx).Query(ctx, updateDepositsStatusSQL, eventTime, exitRoot)
	if err != nil {
		return nil, err
	}

	deposits := make([]*etherman.Deposit, 0, len(rows.RawValues()))
	for rows.Next() {
		var (
			deposit etherman.Deposit
			amount  string
		)
		err = rows.Scan(&deposit.Id, &deposit.LeafType, &deposit.OriginalNetwork, &deposit.OriginalAddress, &amount, &deposit.DestinationNetwork, &deposit.DestinationAddress,
			&deposit.DepositCount, &deposit.BlockID, &deposit.BlockNumber, &deposit.NetworkID, &deposit.TxHash, &deposit.Metadata, &deposit.ReadyForClaim, &deposit.Time)
		if err != nil {
			return nil, err
		}
		deposit.Amount, _ = new(big.Int).SetString(amount, 10) //nolint:gomnd
		deposits = append(deposits, &deposit)
	}
	return deposits, nil
}