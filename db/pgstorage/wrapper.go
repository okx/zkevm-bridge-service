package pgstorage

import (
	"context"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

const (
	defaultDBTimeout = 5 * time.Second
)

// execQuerierWrapper automatically adds a ctx timeout for the querier, also add before and after logs
type execQuerierWrapper struct {
	execQuerier
}

func (w *execQuerierWrapper) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	logger := log.WithFields(utils.TraceID, ctx.Value(utils.TraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[Exec], sql[%v], arguments[%v]", removeNewLine(sql), arguments)

	tag, err := w.execQuerier.Exec(dbCtx, sql, arguments...)

	logger.Debugf("DB query end, method[Exec], err[%v] processTime[%v]", err, time.Since(startTime).String())
	return tag, err
}

func (w *execQuerierWrapper) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	logger := log.WithFields(utils.TraceID, ctx.Value(utils.TraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[Query], sql[%v], arguments[%v]", removeNewLine(sql), args)

	rows, err := w.execQuerier.Query(dbCtx, sql, args...)

	logger.Debugf("DB query end, method[Query], err[%v] processTime[%v]", err, time.Since(startTime).String())
	return rows, err
}

func (w *execQuerierWrapper) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	logger := log.WithFields(utils.TraceID, ctx.Value(utils.TraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[QueryRow], sql[%v], arguments[%v]", removeNewLine(sql), args)

	row := w.execQuerier.QueryRow(dbCtx, sql, args...)

	logger.Debugf("DB query end, method[QueryRow], processTime[%v]", time.Since(startTime).String())
	return row
}

func (w *execQuerierWrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	logger := log.WithFields(utils.TraceID, ctx.Value(utils.TraceID))
	startTime := time.Now()
	logger.Debugf("DB query begin, method[CopyFrom], tableName[%v]", tableName)

	res, err := w.execQuerier.CopyFrom(dbCtx, tableName, columnNames, rowSrc)

	logger.Debugf("DB query end, method[CopyFrom], res[%v] err[%v] processTime[%v]", res, err, time.Since(startTime).String())
	return res, err
}

func getCtxWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, nil
	}
	return context.WithTimeout(ctx, timeout)
}

func removeNewLine(s string) string {
	return strings.Replace(s, "\n", " ", -1)
}