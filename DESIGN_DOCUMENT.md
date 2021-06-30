# TiDB for PgSQL Design Documents

- Author(s): [jiangkun](https://github.com/pupillord)
- Discussion PR: https://github.com/pingcap/tidb/pull/XXX
- Tracking Issue: https://github.com/pingcap/tidb/issues/XXX

## Table of Contents

* [Introduction](#introduction)
* [Motivation or Background](#motivation-or-background)
* [Detailed Design](#detailed-design)
* [Test Design](#test-design)
    * [Functional Tests](#functional-tests)
    * [Scenario Tests](#scenario-tests)
    * [Compatibility Tests](#compatibility-tests)
    * [Benchmark Tests](#benchmark-tests)
* [Impacts & Risks](#impacts--risks)
* [Investigation & Alternatives](#investigation--alternatives)
* [Unresolved Questions](#unresolved-questions)

## Introduction

TiDB for PostgreSQL is an open-source, cloud-native database system developed by Digital China. It aims to provide PostgreSQL compatibility while preserving the high usability, elasticity, and expandability of the TiDB.
We wish it allows users to connect existing PostgreSQL clients to TiDB and uses PostgreSQL-specific syntaxes (Work in progress). 


## Motivation or Background

In the era of the information explosion, centralized databases have slowly become the bottleneck of the entire application system, limiting business growth. Under such circumstances, a series of distributed database systems have emerged, each providing solutions for data migration and expansion.

Among them, TiDB solves most of the migration problems through its high compatibility with MySQL protocols. Furthermore, through its distributed transaction with strong consistency, high expandability, and excellent fault tolerance structure, TiDB convinces users to use its mature solution rather than scale MySQL by sharding. 

However, for businesses based on PostgreSQL, there's no mature solution for migrating to TiDB. If an application wants to move its entire system built on Postgres to TiDB, it has to rewrite most of its business logic to use new features provided by NewSQL. Thus, we approach from the other direction: Make TiDB talks Postgres, thereby making migration to TiDB possible for businesses built on PostgreSQL. 

## Detailed Design

### Overview
As one of the most advanced and feature-rich database systems, PostgreSQL is very complex, and making TiDB fully compatible with PostgreSQL is a mammoth task. Therefore, we decided to divide and conquer. To start simple, we aim to accomplish some simple compatibility first.

In particular, we are currently planning to make TiDB compatible with:

 - PostgreSQL's Communication Protocol
 - Part of PostgreSQL's syntaxes
We will prioritize realizing these  to let PostgreSQL clients connect to TiDB and function normally.


### PostgreSQL Protocol
We start with the communication protocol. TiDB itself uses MySQL protocol, and we need to change it to PostgreSQL protocol. Here we will gloss over the difference in the protocols and focus on the implementation.

There are three common steps between MySQL and PostgreSQL's protocols
1. listening for TCP request and establishing the connection
2. Client-Server handshake to connect successfully
3. Client send request and Server Response

Most of TiDB's communication protocols are implemented under the **Server** package, so we will mostly be modifying this package to make it compatible with the PostgreSQL protocol.

The first step, listening for TCP requests.
This has already been done by TiDB, no code change necessary.

The second step, server-client handshake.
During the establishment of a new connection, TiDB will listen for client TCP connection via methods inside Server/server.go, and verify client through three handshakes. After a successful connection, a Session will be created

    func (s *Server) onConn(conn *clientConn) {  
       ctx := logutil.WithConnID(context.Background(), conn.connectionID)  
       if err := conn.handshake(ctx); err != nil {
       }
       .....
    }

MySQL and PgSQL have different handshaking process, here we will re-implement handshake(ctx) method, complete the PgSQL handshaking request.
Here is an example of the realization: [handshake()](https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL/blob/02ecf71a0a11c5d12f8b6fc823872cdb445f61f4/server/conn.go#L1927)

After the handshake and a successful connection, the rest of the Session info is in sync with the original info from TiDB.

The third step, listening for requests from the Client and returning corresponding responses.

In TiDB, listening for client SQL requests is achieved through **dispatch** method inside **Server/conn.go**. The client sends messages in the form of cmd + data, the server chooses commands based on cmd, and executes based on specifics stored in data.

Messages from PostgreSQL client also includes cmd and data two pieces of information. Thus we can parse the PgSQL message the same way, execute the commands and return the corresponding results to the client. 

    func (cc *clientConn) dispatch(ctx context.Context, data []byte) error {
        ......
        cmd := data[0]  
        data = data[1:]
        ......
        switch cmd {
    	case 'Q':            /* simple query */  
    	   return cc.handleQuery(ctx, dataStr)  
    	case 'P':            /* parse */  
    		return cc.handleStmtPrepare(ctx,parse)  
    	case 'B':            /* bind */  
    		return cc.handleStmtBind(ctx, bind)  
    	case 'E':            /* execute */  
    		return cc.handleStmtExecute(ctx, execute)
    ｝

Here is an example of the realization: [dispatch()](https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL/blob/02ecf71a0a11c5d12f8b6fc823872cdb445f61f4/server/conn.go#L787)

The most important part of this implementation is the format of the messages. The most fundamental difference between messages from MySQL client and PostgreSQL is their byte order -- MySQL uses little-endian and PostgreSQL uses big-endian.
Moreover, the format of the returning message between the two also differs greatly. We realize PostgreSQL message format mainly through 3rd party package: [jackc/pgproto3 (github.com)](https://github.com/jackc/pgproto3)

### PostgreSQL Syntax

The second part is to make TiDB compatible with some of the PostgreSQL syntaxes and keywords. This is similar to how TiDB implements compatibility with MySQL syntaxes. 

Compatibility with different PostgreSQL syntax requires different implementation and different amounts of effort. But most new syntaxes or keywords can be achieved through the following few steps:
 \- Add syntax analysis and generated structure in Parser module
 \- Convert them into the corresponding logical operators and physical operators during execution plan construction
 \- Add possible optimization based on different functional syntaxes
 \- Realize functionality during execution

Adding new keywords and syntax support is already a common procedure within the TiDB community, but this is still a new challenge for us.

The first keyword we plan to work on is **Returning**

     Delete from table_name where col_name = value returning col_name

     Update table_name set col_name = value returning col_name

     Insert into table_name values() returning col_name

Here are two approaches to implement **Returning** keyword

1. Add keyword recognition for Returning in Parser, after query information is passed through, construct a new Select statement. In other words, the original statements will be constructed into two plans during execution -- the original Delete/Update/Inset without returning and a Select, executing both plans inside one transaction and return results. 

2. Similarly, add keyword recognition to Parser. Add a Returning operator when TiDB constructing plans. During the execution of the Delete/Update/Insert, obtain the specified data table in the record, store the info inside Returning operator, filter and return. 

Those are the two approaches we thought of. We hope the PingCAP community can give us some guidance.

We are still searching candidates for new keywords/syntaxes to support. We would like something that has frequent usage, one clear functionality, easy realization, and no conflict with MySQL.

## Hacking Camp 
### Unit Test
We have not successfully pass TiDB unit tests locally yet and haven't written unit tests after modification. We will be adding them in the future.

### Protocol
We have achieved compatibility with basic PostgreSQL protocols, but need a lot of improvements. For example, translation between the PgSQL datatype and TiDB datatype within the protocol, usage of Datum structure, error message conversion between the two systems all still require continuous efforts and improvement.

### Syntax
In this Hacking event, we wish that through guidance from mentors, we can gain a deeper understanding of TiDB's execution framework, and improve our development efficiency. For this, we added some PostgreSQL syntax realization, like Returning keyword.
We plan to make a list of Postgres syntax that we wish to realize and implement as many of them in this event.

## Test Design

We will be mainly focusing on Unit Test and Performance Test

### Functional Tests

- Connection to a PostgreSQL client
- Simple query from PostgreSQL
- Query expansion from PostgreSQL
- Normal usage of additional PostgreSQL syntaxes implemented

### Scenario Tests

N/A

### Compatibility Tests

- Compatibility with TiKV, PD. Can build and deploy a TiDB for PostgreSQL cluster.
- Compatibility with monitor module Dashboard and Garfana
- Compatibility with TiDB execution framework

### Benchmark Tests

Run TiDB for PgSQL and TiDB with different protocols under the same benchmark, look for performance drop and monitor if CPU and memory resource usage is normal.

## Impacts & Risks

- TiDB for PgSQL will primarily focus on supporting PgSQL protocol and syntaxes, it may no longer support MySQL protocol.
- Because the protocol changed to PgSQL, some tools in the TiDB eco-system may stop working. Time will tell. 
- Increased protocol support and a more complex execution process may cause performance degradation. Based on our Sysbench reports earlier, performance will drop under certain scenarios. 

## Investigation & Alternatives

We have considered YugabyteDB and CockroachDB before. They both are NewSQL that claim to supports PostgreSQL, but both still has some compatibility issues. Our attempts of migrating internal systems to them have failed. In the meantime, TiDB is a great open-source project with a thriving community, we hope that TiDB can support PostgreSQL in the future, make it available to more businesses. 

## Unresolved Questions

Making TiDB compatible with PgSQL is a huge challenge for us. There are many features inside TiDB that we wish can be used on PostgreSQL. Rome wasn't built in a day, we plan to realize TiDB for PgSQL step by step, and we start with communication protocol and a small set of syntaxes. 

In the process, we have encountered some problems:

- TiDB and PgSQL uses different data type, how to convert them within the communication protocol
- How can we return normal data sets in plans like Delete/Update/Insert
- How to pass TiDB unit tests and add more unit tests.

There will be more problems during our development, but we are sure that the TiDB community will help us along the way.

