package session

import (
	"context"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/executor"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/auth"
	"github.com/pingcap/tidb/privilege"
)

// PrepareStmt is used for executing prepare statement in binary protocol
// PgSQL Modified
func (s *session) PrepareStmt(sql string, name string) (stmtID uint32, paramCount int, fields []*ast.ResultField, err error) {
	if s.sessionVars.TxnCtx.InfoSchema == nil {
		// We don't need to create a transaction for prepare statement, just get information schema will do.
		s.sessionVars.TxnCtx.InfoSchema = domain.GetDomain(s).InfoSchema()
	}
	err = s.loadCommonGlobalVariablesIfNeeded()
	if err != nil {
		return
	}

	ctx := context.Background()
	inTxn := s.GetSessionVars().InTxn()
	// NewPrepareExec may need startTS to build the executor, for example prepare statement has subquery in int.
	// So we have to call PrepareTxnCtx here.
	s.PrepareTxnCtx(ctx)
	s.PrepareTSFuture(ctx)
	//prepareExec := executor.NewPrepareExec(s, sql)
	prepareExec := executor.NewPrepareExec(s, sql, name)
	err = prepareExec.Next(ctx, nil)
	if err != nil {
		return
	}
	if !inTxn {
		// We could start a transaction to build the prepare executor before, we should rollback it here.
		s.RollbackTxn(ctx)
	}
	return prepareExec.ID, prepareExec.ParamCount, prepareExec.Fields, nil
}

func (s *session) NeedPassword(user *auth.UserIdentity) bool {
	pm := privilege.GetPrivilegeManager(s)
	return pm.NeedPassword(user.Username, user.Hostname)
}
