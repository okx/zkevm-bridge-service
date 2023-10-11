// Code generated by mockery v2.28.1. DO NOT EDIT.

package synchronizer

import (
	context "context"

	etherman "github.com/okx/zkevm-bridge-service/etherman"
	mock "github.com/stretchr/testify/mock"

	pgx "github.com/jackc/pgx/v4"
)

// storageMock is an autogenerated mock type for the storageInterface type
type storageMock struct {
	mock.Mock
}

// AddBlock provides a mock function with given fields: ctx, block, dbTx
func (_m *storageMock) AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error) {
	ret := _m.Called(ctx, block, dbTx)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Block, pgx.Tx) (uint64, error)); ok {
		return rf(ctx, block, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Block, pgx.Tx) uint64); ok {
		r0 = rf(ctx, block, dbTx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *etherman.Block, pgx.Tx) error); ok {
		r1 = rf(ctx, block, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddClaim provides a mock function with given fields: ctx, claim, dbTx
func (_m *storageMock) AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, claim, dbTx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Claim, pgx.Tx) error); ok {
		r0 = rf(ctx, claim, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddDeposit provides a mock function with given fields: ctx, deposit, dbTx
func (_m *storageMock) AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) (uint64, error) {
	ret := _m.Called(ctx, deposit, dbTx)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Deposit, pgx.Tx) (uint64, error)); ok {
		return rf(ctx, deposit, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Deposit, pgx.Tx) uint64); ok {
		r0 = rf(ctx, deposit, dbTx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *etherman.Deposit, pgx.Tx) error); ok {
		r1 = rf(ctx, deposit, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddGlobalExitRoot provides a mock function with given fields: ctx, exitRoot, dbTx
func (_m *storageMock) AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, exitRoot, dbTx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.GlobalExitRoot, pgx.Tx) error); ok {
		r0 = rf(ctx, exitRoot, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddTokenWrapped provides a mock function with given fields: ctx, tokenWrapped, dbTx
func (_m *storageMock) AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, tokenWrapped, dbTx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.TokenWrapped, pgx.Tx) error); ok {
		r0 = rf(ctx, tokenWrapped, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddTrustedGlobalExitRoot provides a mock function with given fields: ctx, trustedExitRoot, dbTx
func (_m *storageMock) AddTrustedGlobalExitRoot(ctx context.Context, trustedExitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) (bool, error) {
	ret := _m.Called(ctx, trustedExitRoot, dbTx)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.GlobalExitRoot, pgx.Tx) (bool, error)); ok {
		return rf(ctx, trustedExitRoot, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.GlobalExitRoot, pgx.Tx) bool); ok {
		r0 = rf(ctx, trustedExitRoot, dbTx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *etherman.GlobalExitRoot, pgx.Tx) error); ok {
		r1 = rf(ctx, trustedExitRoot, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BeginDBTransaction provides a mock function with given fields: ctx
func (_m *storageMock) BeginDBTransaction(ctx context.Context) (pgx.Tx, error) {
	ret := _m.Called(ctx)

	var r0 pgx.Tx
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (pgx.Tx, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) pgx.Tx); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(pgx.Tx)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Commit provides a mock function with given fields: ctx, dbTx
func (_m *storageMock) Commit(ctx context.Context, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, dbTx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) error); ok {
		r0 = rf(ctx, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetLastBlock provides a mock function with given fields: ctx, networkID, dbTx
func (_m *storageMock) GetLastBlock(ctx context.Context, networkID uint, dbTx pgx.Tx) (*etherman.Block, error) {
	ret := _m.Called(ctx, networkID, dbTx)

	var r0 *etherman.Block
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, pgx.Tx) (*etherman.Block, error)); ok {
		return rf(ctx, networkID, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint, pgx.Tx) *etherman.Block); ok {
		r0 = rf(ctx, networkID, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*etherman.Block)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint, pgx.Tx) error); ok {
		r1 = rf(ctx, networkID, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestL1SyncedExitRoot provides a mock function with given fields: ctx, dbTx
func (_m *storageMock) GetLatestL1SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error) {
	ret := _m.Called(ctx, dbTx)

	var r0 *etherman.GlobalExitRoot
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) (*etherman.GlobalExitRoot, error)); ok {
		return rf(ctx, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) *etherman.GlobalExitRoot); ok {
		r0 = rf(ctx, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*etherman.GlobalExitRoot)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx) error); ok {
		r1 = rf(ctx, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNumberDeposits provides a mock function with given fields: ctx, origNetworkID, blockNumber, dbTx
func (_m *storageMock) GetNumberDeposits(ctx context.Context, origNetworkID uint, blockNumber uint64, dbTx pgx.Tx) (uint64, error) {
	ret := _m.Called(ctx, origNetworkID, blockNumber, dbTx)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint64, pgx.Tx) (uint64, error)); ok {
		return rf(ctx, origNetworkID, blockNumber, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint64, pgx.Tx) uint64); ok {
		r0 = rf(ctx, origNetworkID, blockNumber, dbTx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint, uint64, pgx.Tx) error); ok {
		r1 = rf(ctx, origNetworkID, blockNumber, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPreviousBlock provides a mock function with given fields: ctx, networkID, offset, dbTx
func (_m *storageMock) GetPreviousBlock(ctx context.Context, networkID uint, offset uint64, dbTx pgx.Tx) (*etherman.Block, error) {
	ret := _m.Called(ctx, networkID, offset, dbTx)

	var r0 *etherman.Block
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint64, pgx.Tx) (*etherman.Block, error)); ok {
		return rf(ctx, networkID, offset, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint64, pgx.Tx) *etherman.Block); ok {
		r0 = rf(ctx, networkID, offset, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*etherman.Block)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint, uint64, pgx.Tx) error); ok {
		r1 = rf(ctx, networkID, offset, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Reset provides a mock function with given fields: ctx, blockNumber, networkID, dbTx
func (_m *storageMock) Reset(ctx context.Context, blockNumber uint64, networkID uint, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, blockNumber, networkID, dbTx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, uint, pgx.Tx) error); ok {
		r0 = rf(ctx, blockNumber, networkID, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rollback provides a mock function with given fields: ctx, dbTx
func (_m *storageMock) Rollback(ctx context.Context, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, dbTx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) error); ok {
		r0 = rf(ctx, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTnewStorageMock interface {
	mock.TestingT
	Cleanup(func())
}

// newStorageMock creates a new instance of storageMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newStorageMock(t mockConstructorTestingTnewStorageMock) *storageMock {
	mock := &storageMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
