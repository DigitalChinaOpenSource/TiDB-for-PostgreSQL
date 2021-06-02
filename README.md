# TiDB for postgresql

## Introduction

TiDB for postgresql is an open source database product launched by Digital China Cloud Base to promote and integrate into the ecological activities of the TiDB open source community. In addition to its high availability, elastic scalability, and cloud-native features, TiDB for postgresql is biggest feature is its compatibility with the postgresql protocol. On the one hand, you can use the postgresql client to connect to TiDB for postgresql, which is the function we have basically completed. On the other hand, you can also use the unique grammatical features of postgresql in your business system, which is also a function we are currently in the development stage.



## Background

As the general trend of the massive explosion of information in the current data era becomes more and more obvious, the database that needs to support the business system is more and more likely to become the bottleneck of the upper-level business system. For this problem, TiDB solves most of migration problems of business system based on MySQL database through the highly compatible MySQL protocol. At the same time, through its strong distributed consistency, high scalability, and excellent disaster tolerance backup architecture, users abandon the idea of extending MySQL by sub-database and sub-table. Use TiDB's mature solutions directly. However, there is no mature solution for the upper-level business system based on postgresql to migrate to TiDB. If you want to migrate a postgresql-based business system to TiDB and use the features of distributed New SQL, you will have to face a huge workload of business system code modification. So we started from another aspect. It made it possible to migrate many postgresql-based business systems to TiDB for postgresql by refactoring the underlying source code of TiDB to make it compatible with the postgresql protocol.



## Development progress

The biggest workload of TiDB for postgresql is compatibility with postgresql. This has two implications. One is to implement the postgresql connection protocol, which we have basically achieved. Postgresql clients can connect to TiDB for postgresql. The other is the postgresql syntax feature. TiDB for postgresql can support regular sql syntax, but it requires a certain amount of work to be compatible with the MySQL-specific syntax of postgresql. This is one of the reasons why TiDB for postgresql is open sourced. We hope that through the power of open source, many technical personnel interested in open source databases will work together to make this project bigger and better.



## Quick start

First make sure to install the go environment, because TiDB for postgresql is based on the go language.

TiDB for postgresql can be started on a single node without pd and tikv.

If there is no pd and tikv, it will create mock pd and mock pd to maintain the stable operation of the system

The following is an example of locally compiling and running TiDB for postgresql on localhost

```shell
mkdir -p  $GOPATH/src/github.com/digitalchina

cd  $GOPATH/src/github.com/digitalchina

git clone 地址待定

cd tidb/tidb-server

go run main.go
```

After starting the main program of TiDB for postgresql , it will run on port 4000 of the host.

Use postgresql client to connect to TiDB for postgresql. Here, take the command line tool psql that comes with postgresql as an example to connect to TiDB for postgresql running on this machine.

```
Server [localhost]:
Database [postgres]: test
Port [5433]: 4000
Username [postgres]: root
psql (13.1, server 8.3.11)
Type "help" to get help information.

test=# show tables;
 Tables_in_test
----------------
 t1
 t2
(2 行记录)
```

TiDB for postgresql also supports cluster deployment
Our current modification work does not involve the communication module of each component in the cluster. Therefore, the function of TiDB for postgresql connecting to pd and tikv will not be affected in any way

We recommend using binary file to deploy TiDB for postgresql cluster.

First, download the official binary package file and unzip it.

```shell
wget http://download.pingcap.org/tidb-latest-linux-amd64.tar.gz
wget http://download.pingcap.org/tidb-latest-linux-amd64.sha256

sha256sum -c tidb-latest-linux-amd64.sha256

tar -xzf tidb-latest-linux-amd64.tar.gz
cd tidb-latest-linux-amd64/bin
```

The second step is to deploy each node in the cluster in order. According to the TiDB for postgresql cluster architecture, pd nodes are deployed first. Then deploy the tikv node, and finally deploy the TiDB for postgresql node. The number of nodes in the cluster can be arbitrarily matched, and there is at least one.

Deploy one pd node

```shell
./pd-server --name=pd1 --data-dir=pd1 --client-urls="http://pdhost:2379" --peer-urls="http://host:2380" -L "info" --log-file=pd.log
```

Deploy three tikv nodes

```shell
./tikv-server --pd="pdhost:2379" --addr="kvhost1:20160"  --data-dir=tikv  --log-file=tikv1.log
./tikv-server --pd="pdhost:2379" --addr="kvhost2:20160"  --data-dir=tikv  --log-file=tikv2.log
./tikv-server --pd="pdhost:2379" --addr="kvhost3:20160"  --data-dir=tikv  --log-file=tikv3.log
```

Deploy one TiDB for postgresql node. You need to compile the TiDB for postgresql project into a binary file named tidb-server and upload it to this server

```shell
./tidb-server --store=tikv  --path="pdhost:2379" --log-file=tidb.log
```

After the above operations, a TiDB for postgresql cluster is successfully deployed, and you can now connect to TiDB for postgresql in exactly the same way as the previous demonstration of connecting to the stand-alone TiDB for postgresql.







## Contribute code to TiDB for postgresql

We very much welcome developers who are interested in TiDB for postgresql's postgresql compatibility to contribute code to this project. At present, we provide several entry points to get started.

### Schema structure

The database structure of postgressql is very different from that of MySQL, which is also a big problem we encountered in the development process. It involves the realization of database system tables, system views, and system functions. The workload is very large, and the difficulty of realization is not small. Developers are required to be very familiar with the database table structure of posygresql and MySQL at the same time.

### Postgresql grammatical features

Compared with MySQL, posrgresql has many new syntax features, such as Returning can return modified row data. Implementing a grammatical feature involves modifying the Parser module, TiDB for postgresql's internal plan structure, plan optimization, plan execution, and data write-back, and many other parts of the code. The difficulty can be said to be huge. It is recommended that developers start with clauses such as RETURNING and modify them on the basis of the original code. After being familiar with the logic of TiDB for postgresql planning and execution, you can try to implement a brand new statement implementation.



## Version notice

For the stable progress of the code transformation work, we currently choose a fixed version for development. The following is our selected version:
TiDB v4.0.11
postgresql 13



## License

开源证书待定



