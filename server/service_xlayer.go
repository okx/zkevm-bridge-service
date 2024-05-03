package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/estimatetime"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/localcache"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/pushtask"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/pkg/errors"
)

const (
	mtHeight           = 32 // For sending mtProof to bridge contract, it requires constant-sized array...
	defaultMinDuration = 1
)

var (
	minReadyTimeLimitForWaitClaimSeconds = apolloconfig.NewIntEntry[int64]("api.minReadyTimeLimitForWaitClaim", 24*60*1000) //nolint:gomnd
)

func (s *bridgeService) WithRedisStorage(storage redisstorage.RedisStorage) *bridgeService {
	s.redisStorage = storage
	return s
}

func (s *bridgeService) WithMainCoinsCache(cache localcache.MainCoinsCache) *bridgeService {
	s.mainCoinsCache = cache
	return s
}

func (s *bridgeService) WithMessagePushProducer(producer messagepush.KafkaProducer) *bridgeService {
	s.messagePushProducer = producer
	return s
}

func (s *bridgeService) GetSmtProof(ctx context.Context, req *pb.GetSmtProofRequest) (*pb.CommonProofResponse, error) {
	globalExitRoot, merkleProof, rollupMerkleProof, err := s.GetClaimProof(uint(req.Index), uint(req.FromChain), nil)
	if err != nil || len(merkleProof) != len(rollupMerkleProof) {
		log.Errorf("GetSmtProof err[%v] merkleProofLen[%v] rollupMerkleProofLen[%v]", err, len(merkleProof), len(rollupMerkleProof))
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, err.Error())
	}
	var (
		proof       []string
		rollupProof []string
	)
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
		rollupProof = append(rollupProof, "0x"+hex.EncodeToString(rollupMerkleProof[i][:]))
	}

	return &pb.CommonProofResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.ProofDetail{
			SmtProof:        proof,
			RollupSmtProof:  rollupProof,
			MainnetExitRoot: globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot:  globalExitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetCoinPrice returns the price for each coin symbol in the request
// Bridge rest API endpoint
func (s *bridgeService) GetCoinPrice(ctx context.Context, req *pb.GetCoinPriceRequest) (*pb.CommonCoinPricesResponse, error) {
	// convert inner chainId to standard chain id
	for _, symbol := range req.SymbolInfos {
		symbol.ChainId = utils.GetStandardChainIdByInnerId(symbol.ChainId)
	}
	priceList, err := s.redisStorage.GetCoinPrice(ctx, req.SymbolInfos)
	if err != nil {
		log.Errorf("get coin price from redis failed for symbol: %v, error: %v", req.SymbolInfos, err)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, gerror.ErrInternalErrorForRpcCall.Error())
	}
	// convert standard chainId to ok inner chainId
	for _, priceInfo := range priceList {
		priceInfo.ChainId = utils.GetInnerChainIdByStandardId(priceInfo.ChainId)
	}
	return &pb.CommonCoinPricesResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: priceList,
	}, nil
}

// GetMainCoins returns the info of the main coins in a network
// Bridge rest API endpoint
func (s *bridgeService) GetMainCoins(ctx context.Context, req *pb.GetMainCoinsRequest) (*pb.CommonCoinsResponse, error) {
	coins, err := s.mainCoinsCache.GetMainCoinsByNetwork(ctx, req.NetworkId)
	if err != nil {
		log.Errorf("get main coins from cache failed for net: %v, error: %v", req.NetworkId, err)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, gerror.ErrInternalErrorForRpcCall.Error())
	}
	// use ok inner chain id
	for _, coinInfo := range coins {
		coinInfo.ChainId = utils.GetInnerChainIdByStandardId(coinInfo.ChainId)
	}
	return &pb.CommonCoinsResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: coins,
	}, nil
}

// GetPendingTransactions returns the pending transactions of an account
// Bridge rest API endpoint
func (s *bridgeService) GetPendingTransactions(ctx context.Context, req *pb.GetPendingTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}

	deposits, err := s.storage.GetPendingTransactions(ctx, req.DestAddr, uint(limit+1), uint(req.Offset), utils.GetUSDCContractAddressList(), nil)
	if err != nil {
		log.Errorf("get pending tx failed for address: %v, limit: %v, offset: %v, error: %v", req.DestAddr, limit, req.Offset, err)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, gerror.ErrInternalErrorForRpcCall.Error())
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[:limit]
	}

	l1BlockNum, _ := s.redisStorage.GetL1BlockNum(ctx)
	l2CommitBlockNum, _ := s.redisStorage.GetCommitMaxBlockNum(ctx)
	l2AvgCommitDuration := pushtask.GetAvgCommitDuration(ctx, s.redisStorage)
	l2AvgVerifyDuration := pushtask.GetAvgVerifyDuration(ctx, s.redisStorage)
	currTime := time.Now()

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		// replace contract address to real token address
		utils.ReplaceUSDCDepositInfo(deposit, false)
		transaction := utils.EthermanDepositToPbTransaction(deposit)
		transaction.EstimateTime = estimatetime.GetDefaultCalculator().Get(deposit.NetworkID)
		transaction.Status = uint32(pb.TransactionStatus_TX_CREATED)
		transaction.GlobalIndex = s.getGlobalIndex(deposit).String()
		if deposit.ReadyForClaim {
			transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM)
			// For L1->L2, if backend is trying to auto-claim, set the status to 0 to block the user from manual-claim
			// When the auto-claim failed, set status to 1 to let the user claim manually through front-end
			if deposit.NetworkID == 0 {
				mTx, err := s.storage.GetClaimTxById(ctx, deposit.DepositCount, nil)
				if err == nil && mTx.Status != ctmtypes.MonitoredTxStatusFailed {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
				}
			}
		} else {
			// For L1->L2, when ready_for_claim is false, but there have been more than 64 block confirmations,
			// should also display the status as "L2 executing" (pending auto claim)
			if deposit.NetworkID == 0 {
				if l1BlockNum-deposit.BlockNumber >= utils.L1TargetBlockConfirmations.Get() {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
				}
			} else {
				if l2CommitBlockNum >= deposit.BlockNumber {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_VERIFICATION)
				}
				s.setDurationForL2Deposit(ctx, l2AvgCommitDuration, l2AvgVerifyDuration, currTime, transaction, deposit.Time)
			}
		}
		// chain id convert to ok inner chain id
		if transaction.FromChainId != 0 {
			transaction.FromChainId = uint32(utils.GetInnerChainIdByStandardId(uint64(transaction.FromChainId)))
		}
		if transaction.ToChainId != 0 {
			transaction.ToChainId = uint32(utils.GetInnerChainIdByStandardId(uint64(transaction.ToChainId)))
		}
		pbTransactions = append(pbTransactions, transaction)
	}
	return &pb.CommonTransactionsResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

func (s *bridgeService) setDurationForL2Deposit(ctx context.Context, l2AvgCommitDuration uint64, l2AvgVerifyDuration uint64, currTime time.Time,
	tx *pb.Transaction, depositCreateTime time.Time) {
	var duration int
	if tx.Status == uint32(pb.TransactionStatus_TX_CREATED) {
		duration = pushtask.GetLeftCommitTime(depositCreateTime, l2AvgCommitDuration, currTime)
	} else {
		duration = pushtask.GetLeftVerifyTime(ctx, s.redisStorage, tx.BlockNumber, depositCreateTime, l2AvgCommitDuration, l2AvgVerifyDuration, currTime)
	}
	if duration <= 0 {
		log.Debugf("count EstimateTime for L2 -> L1 over range, so use min default duration: %v", defaultMinDuration)
		tx.EstimateTime = uint32(defaultMinDuration)
		return
	}
	tx.EstimateTime = uint32(duration)
}

// GetAllTransactions returns all the transactions of an account, similar to GetBridges
// Bridge rest API endpoint
func (s *bridgeService) GetAllTransactions(ctx context.Context, req *pb.GetAllTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}

	deposits, err := s.storage.GetDepositsXLayer(ctx, req.DestAddr, uint(limit+1), uint(req.Offset), utils.GetUSDCContractAddressList(), nil)
	if err != nil {
		log.Errorf("get deposits from db failed for address: %v, limit: %v, offset: %v, error: %v", req.DestAddr, limit, req.Offset, err)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, gerror.ErrInternalErrorForRpcCall.Error())
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[0:limit]
	}

	l1BlockNum, _ := s.redisStorage.GetL1BlockNum(ctx)
	l2CommitBlockNum, _ := s.redisStorage.GetCommitMaxBlockNum(ctx)
	l2AvgCommitDuration := pushtask.GetAvgCommitDuration(ctx, s.redisStorage)
	l2AvgVerifyDuration := pushtask.GetAvgVerifyDuration(ctx, s.redisStorage)
	currTime := time.Now()

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		// replace contract address to real token address
		utils.ReplaceUSDCDepositInfo(deposit, false)
		transaction := utils.EthermanDepositToPbTransaction(deposit)
		transaction.EstimateTime = estimatetime.GetDefaultCalculator().Get(deposit.NetworkID)
		transaction.Status = uint32(pb.TransactionStatus_TX_CREATED) // Not ready for claim
		transaction.GlobalIndex = s.getGlobalIndex(deposit).String()
		if deposit.ReadyForClaim {
			// Check whether it has been claimed or not
			claim, err := s.storage.GetClaim(ctx, deposit.DepositCount, deposit.DestinationNetwork, nil)
			transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM) // Ready but not claimed
			if err != nil {
				if !errors.Is(err, gerror.ErrStorageNotFound) {
					return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, fmt.Sprintf("load claim error: %v", err))
				}
				// For L1->L2, if backend is trying to auto-claim, set the status to 0 to block the user from manual-claim
				// When the auto-claim failed, set status to 1 to let the user claim manually through front-end
				if deposit.NetworkID == 0 {
					mTx, err := s.storage.GetClaimTxById(ctx, deposit.DepositCount, nil)
					if err == nil && mTx.Status != ctmtypes.MonitoredTxStatusFailed {
						transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
					}
				}
			} else {
				transaction.Status = uint32(pb.TransactionStatus_TX_CLAIMED) // Claimed
				transaction.ClaimTxHash = claim.TxHash.String()
				transaction.ClaimTime = uint64(claim.Time.UnixMilli())
			}
		} else {
			// For L1->L2, when ready_for_claim is false, but there have been more than 64 block confirmations,
			// should also display the status as "L2 executing" (pending auto claim)
			if deposit.NetworkID == 0 {
				if l1BlockNum-deposit.BlockNumber >= utils.L1TargetBlockConfirmations.Get() {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
				}
			} else {
				if l2CommitBlockNum >= deposit.BlockNumber {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_VERIFICATION)
				}
				s.setDurationForL2Deposit(ctx, l2AvgCommitDuration, l2AvgVerifyDuration, currTime, transaction, deposit.Time)
			}
		}
		// chain id convert to ok inner chain id
		if transaction.FromChainId != 0 {
			transaction.FromChainId = uint32(utils.GetInnerChainIdByStandardId(uint64(transaction.FromChainId)))
		}
		if transaction.ToChainId != 0 {
			transaction.ToChainId = uint32(utils.GetInnerChainIdByStandardId(uint64(transaction.ToChainId)))
		}
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonTransactionsResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

// GetNotReadyTransactions returns all deposit transactions with ready_for_claim = false
func (s *bridgeService) GetNotReadyTransactions(ctx context.Context, req *pb.GetNotReadyTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}

	deposits, err := s.storage.GetNotReadyTransactions(ctx, uint(limit+1), uint(req.Offset), nil)
	if err != nil {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, err.Error())
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[0:limit]
	}

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		transaction := utils.EthermanDepositToPbTransaction(deposit)
		transaction.EstimateTime = estimatetime.GetDefaultCalculator().Get(deposit.NetworkID)
		transaction.Status = uint32(pb.TransactionStatus_TX_CREATED)
		transaction.GlobalIndex = s.getGlobalIndex(deposit).String()
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonTransactionsResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

// GetMonitoredTxsByStatus returns list of monitored transactions, filtered by status
func (s *bridgeService) GetMonitoredTxsByStatus(ctx context.Context, req *pb.GetMonitoredTxsByStatusRequest) (*pb.CommonMonitoredTxsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}

	mTxs, err := s.storage.GetClaimTxsByStatusWithLimit(ctx, []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatus(req.Status)}, uint(limit+1), uint(req.Offset), nil)
	if err != nil {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, err.Error())
	}

	hasNext := len(mTxs) > int(limit)
	if hasNext {
		mTxs = mTxs[0:limit]
	}

	var pbTransactions []*pb.MonitoredTx
	for _, mTx := range mTxs {
		transaction := &pb.MonitoredTx{
			Id:        uint64(mTx.DepositID),
			From:      "0x" + mTx.From.String(),
			To:        "0x" + mTx.To.String(),
			Nonce:     mTx.Nonce,
			Value:     mTx.Value.String(),
			Data:      "0x" + hex.EncodeToString(mTx.Data),
			Gas:       mTx.Gas,
			GasPrice:  mTx.GasPrice.String(),
			Status:    string(mTx.Status),
			CreatedAt: uint64(mTx.CreatedAt.UnixMilli()),
			UpdatedAt: uint64(mTx.UpdatedAt.UnixMilli()),
		}
		for h := range mTx.History {
			transaction.History = append(transaction.History, h.String())
		}
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonMonitoredTxsResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.MonitoredTxsDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

// GetEstimateTime returns the estimated deposit waiting time for L1 and L2
func (s *bridgeService) GetEstimateTime(ctx context.Context, req *pb.GetEstimateTimeRequest) (*pb.CommonEstimateTimeResponse, error) {
	return &pb.CommonEstimateTimeResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: []uint32{estimatetime.GetDefaultCalculator().Get(0), estimatetime.GetDefaultCalculator().Get(1)},
	}, nil
}

// ManualClaim manually sends a claim transaction for a specific deposit
func (s *bridgeService) ManualClaim(ctx context.Context, req *pb.ManualClaimRequest) (*pb.CommonManualClaimResponse, error) {
	// Only allow L1->L2
	if req.FromChain != 0 {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "only allow L1->L2 claim")
	}

	// Query the deposit info from storage
	deposit, err := s.storage.GetDepositByHash(ctx, req.DestAddr, uint(req.FromChain), req.DepositTxHash, nil)
	if err != nil {
		log.Errorf("Failed to get deposit: %v", err)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "failed to get deposit info")
	}

	// Only allow to claim ready transactions
	if !deposit.ReadyForClaim {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "transaction is not ready for claim")
	}

	// Check whether the deposit has already been claimed
	_, err = s.storage.GetClaim(ctx, deposit.DepositCount, deposit.DestinationNetwork, nil)
	if err == nil {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "transaction has already been claimed")
	}
	if !errors.Is(err, gerror.ErrStorageNotFound) {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, err.Error())
	}

	destNet := deposit.DestinationNetwork
	client, ok := s.nodeClients[destNet]
	if !ok || client == nil {
		log.Errorf("node client for networkID %v not found", destNet)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "")
	}
	// Get the claim proof
	ger, proves, rollupProves, err := s.GetClaimProof(deposit.DepositCount, deposit.NetworkID, nil)
	if err != nil {
		log.Errorf("failed to get claim proof for deposit %v networkID %v: %v", deposit.DepositCount, deposit.NetworkID, err)
	}
	var (
		mtProves       [mtHeight][bridgectrl.KeyLen]byte
		mtRollupProves [mtHeight][bridgectrl.KeyLen]byte
	)
	for i := 0; i < mtHeight; i++ {
		mtProves[i] = proves[i]
		mtRollupProves[i] = rollupProves[i]
	}
	// Send claim transaction to the node
	tx, err := client.SendClaimXLayer(ctx, deposit, mtProves, mtRollupProves, ger, s.rollupID, s.auths[destNet])
	if err != nil {
		log.Errorf("failed to send claim transaction: %v", err)
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "failed to send claim transaction")
	}

	return &pb.CommonManualClaimResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.ManualClaimResponse{
			ClaimTxHash: tx.Hash().String(),
		},
	}, nil
}

// GetReadyPendingTransactions returns all transactions from a network which are ready_for_claim but not claimed
func (s *bridgeService) GetReadyPendingTransactions(ctx context.Context, req *pb.GetReadyPendingTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}

	minReadyTime := time.Now().Add(time.Duration(-minReadyTimeLimitForWaitClaimSeconds.Get()) * time.Second)

	deposits, err := s.storage.GetReadyPendingTransactions(ctx, uint(req.NetworkId), uint(limit+1), uint(req.Offset), minReadyTime, nil)
	if err != nil {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, err.Error())
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[:limit]
	}

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		transaction := utils.EthermanDepositToPbTransaction(deposit)
		transaction.EstimateTime = estimatetime.GetDefaultCalculator().Get(deposit.NetworkID)
		transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM)
		transaction.GlobalIndex = s.getGlobalIndex(deposit).String()
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonTransactionsResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

func (s *bridgeService) getGlobalIndex(deposit *etherman.Deposit) *big.Int {
	mainnetFlag := deposit.NetworkID == 0
	rollupIndex := s.rollupID - 1
	localExitRootIndex := deposit.DepositCount
	return etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
}

func (s *bridgeService) GetFakePushMessages(ctx context.Context, req *pb.GetFakePushMessagesRequest) (*pb.GetFakePushMessagesResponse, error) {
	if s.messagePushProducer == nil {
		return nil, NewStatusError(pb.ErrorCode_ERROR_DEFAULT, "producer is nil")
	}

	return &pb.GetFakePushMessagesResponse{
		Code: uint32(pb.ErrorCode_ERROR_OK),
		Data: s.messagePushProducer.GetFakeMessages(req.Topic),
	}, nil
}