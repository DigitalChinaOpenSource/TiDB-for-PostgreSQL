// Copyright 2021 Digital China Group Co.,Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"context"
	"runtime/trace"
	"time"

	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/execdetails"
)

type ReturningExec struct {
	baseExecutor

	Idx     int
	fetched bool
	schema  *expression.Schema

	ResultSet *recordSet
}

func (e *ReturningExec) Open(ctx context.Context) error {

	return e.children[0].Open(ctx)
}

func (e *ReturningExec) Next(ctx context.Context, req *chunk.Chunk) error {
	return e.fetchRowChunks(ctx)
}

func (e *ReturningExec) Close() error {
	return e.children[0].Close()
}

func (e *ReturningExec) fetchRowChunks(ctx context.Context) error {
	defer func() {
		e.fetched = true
	}()

	var stmtDetail *execdetails.StmtExecDetails
	stmtDetailRaw := ctx.Value(execdetails.StmtExecDetailKey)
	if stmtDetailRaw != nil {
		stmtDetail = stmtDetailRaw.(*execdetails.StmtExecDetails)
	}

	rs := &recordSet{
		executor: e.base().children[0],
	}

	rs.rows = make([]chunk.Row, 0, 1024)

	for {
		req := rs.NewChunk()
		// Here server.tidbResultSet implements Next method.
		err := rs.Next(ctx, req)
		if err != nil {
			return err
		}

		rowCount := req.NumRows()
		if rowCount == 0 {
			break
		}
		start := time.Now()
		reg := trace.StartRegion(ctx, "ProcessReturning")

		for i := 0; i < rowCount; i++ {
			row := req.GetRow(i)

			row.IsNull(0)

			rs.rows = append(rs.rows, row)
		}

		if stmtDetail != nil {
			stmtDetail.WriteSQLRespDuration += time.Since(start)
		}
		reg.End()
	}

	e.ResultSet = rs

	return nil
}
