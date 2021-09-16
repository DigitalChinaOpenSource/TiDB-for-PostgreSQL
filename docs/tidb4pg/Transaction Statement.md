# Adding Support for Transaction Statement in DCParser

---

### PostgreSQL Transaction Statement

#### BEGIN

```
BEGIN [ WORK | TRANSACTION ] [ transaction_mode [, ...] ]

where transaction_mode is one of:

    ISOLATION LEVEL { SERIALIZABLE | REPEATABLE READ | READ COMMITTED | READ UNCOMMITTED }
    READ WRITE | READ ONLY
    [ NOT ] DEFERRABLE
```

#### SET TRANSACTION

The `SET TRANSACTION` command sets the characteristics of the current transaction. It has no effect on any subsequent transactions. `SET SESSION CHARACTERISTICS` sets the default transaction characteristics for subsequent transactions of a session. These defaults can be overridden by `SET TRANSACTION` for an individual transaction.

```
SET TRANSACTION transaction_mode [, ...]
SET TRANSACTION SNAPSHOT snapshot_id
SET SESSION CHARACTERISTICS AS TRANSACTION transaction_mode [, ...]

where transaction_mode is one of:

    ISOLATION LEVEL { SERIALIZABLE | REPEATABLE READ | READ COMMITTED | READ UNCOMMITTED }
    READ WRITE | READ ONLY
    [ NOT ] DEFERRABLE
```



### MySQL Transaction Statement

#### BEGIN / START TRANSACTION

```
START TRANSACTION
    [transaction_characteristic [, transaction_characteristic] ...]

transaction_characteristic: {
    WITH CONSISTENT SNAPSHOT
  | READ WRITE
  | READ ONLY
}

BEGIN [WORK]
COMMIT [WORK] [AND [NO] CHAIN] [[NO] RELEASE]
ROLLBACK [WORK] [AND [NO] CHAIN] [[NO] RELEASE]
SET autocommit = {0 | 1}
```



#### SET TRANSACTION

```
SET [GLOBAL | SESSION] TRANSACTION
    transaction_characteristic [, transaction_characteristic] ...

transaction_characteristic: {
    ISOLATION LEVEL level
  | access_mode
}

level: {
     REPEATABLE READ
   | READ COMMITTED
   | READ UNCOMMITTED
   | SERIALIZABLE
}

access_mode: {
     READ WRITE
   | READ ONLY
}
```

SET TRANSACTION (without specifying scope), only works for the **next** transaction, and cannot be used during transaction

```
mysql> START TRANSACTION;
Query OK, 0 rows affected (0.02 sec)

mysql> SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
ERROR 1568 (25001): Transaction characteristics can't be changed
while a transaction is in progress
```

### TiDB

TiDB only actually supports `Read Committed` and `Repeatable Read` (`Snapshot Isolation`).

And to start a transaction in Read Committed, the transaction must be in pessimistic transaction mode

> The Read Committed isolation level only takes effect in the [pessimistic transaction mode](https://docs.pingcap.com/tidb/stable/pessimistic-transaction). In the [optimistic transaction mode](https://docs.pingcap.com/tidb/stable/optimistic-transaction), setting the transaction isolation level to `Read Committed` does not take effect and transactions still use the Repeatable Read isolation level.

https://docs.pingcap.com/zh/tidb/stable/sql-statement-set-transaction#set-transaction

https://docs.pingcap.com/zh/tidb/stable/transaction-isolation-levels

> Note that TiDB's definition for `Snapshot Isolation` and `Read Committed` is different from ANSI standard and MySQL

### Modification Plan

#### Adding Parser Supports for `BEGIN ISOLATION LEVEL`

In yacc file:

```
|	"BEGIN" "ISOLATION" "LEVEL" IsolationLevel
	{
		$$ = &ast.BeginStmt{
			IsolationLevel: $4,
		}
	}
```

Where IsolationLevel is a non-terminal used by `SET` statement that resolve actual isolation level strings.

Then we have to change `BeginStmt` node structure in AST tree, so it can carry the isolation level we resolved.

```
type BeginStmt struct {
	// Other stuff
	IsolationLevel string
}
```

We also have to change the corresponding restore function to pass the Unit Tests

``` go
func (n *BeginStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Mode == "" {
		if n.ReadOnly {
				// read only stuff
				} else {
			ctx.WriteKeyWord("START TRANSACTION")
			if n.IsolationLevel != "" {
				switch n.IsolationLevel {
				case ReadCommitted:
					ctx.WriteKeyWord(" ISOLATION LEVEL READ COMMITTED")
				case ReadUncommitted:
					ctx.WriteKeyWord(" ISOLATION LEVEL READ UNCOMMITTED")
				case Serializable:
					ctx.WriteKeyWord(" ISOLATION LEVEL SERIALIZABLE")
				case RepeatableRead:
					ctx.WriteKeyWord(" ISOLATION LEVEL REPEATABLE READ")
				}

			}
		}
	} else {
		// begin with mode
	}
	return nil
}
```

Note that since we are restoring it to "`START TRANSACTION ISOLATION LEVEL` ...", we have to add similar yacc rule for `START TRANSACTION`, even though we are not using it. 

```
|	"START" "TRANSACTION" "ISOLATION" "LEVEL" IsolationLevel
	{
		$$ = &ast.BeginStmt{
			IsolationLevel: $5,
		}
	}
```



### Change Begin Executor's behavior in TiDB

All that's left to do is to set transaction level when starting the transaction. In `executeBegin` function under `executor/simple.go`:

``` go
func (e *SimpleExec) executeBegin(ctx context.Context, s *ast.BeginStmt) error {
// ...

	// if the isolation level is set during the transaction start
	// it will overwrite the previously set isolation level
	if s.IsolationLevel != "" {
		e.ctx.GetSessionVars().TxnCtx.Isolation = s.IsolationLevel
	}

// ...
}
```

Note that here we are putting at the bottom of the execution sequence, since it should take the highest priority. 

MySQL (Or Original TiDB), has six layer of isolation level definition, and they take on different values. The priority from low to high is here:

1. Value defined in the configuration file(s).
2. Value used in the command line option used to start *`mysqld`*.
3. The global transaction isolation level.
4. The session transaction isolation level.
5. The level that will be used by the very next transaction that is created.
6. The level being used by the current transaction.

The `begin` statement actually directly set the isolation at the 6th layer, which has the highest priority.

### Validation

Verifying the effect of setting transaction level in begin statement is actually quite problematic. There is no way to view isolation level set in level 5 and 6 in MySQL, and TiDB inherits that.

> This is actually considered a bug: https://bugs.mysql.com/bug.php?id=53341

So the only way to access the value is through integrated test or viewing value directly through debugging tools. 

