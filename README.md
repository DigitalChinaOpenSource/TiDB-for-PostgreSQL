# TiDB for postgresql

[![LICENSE](https://img.shields.io/github/license/pingcap/tidb.svg)](https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL/blob/main/LICENSE)
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Build Status](https://travis-ci.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL.svg?branch=main)](https://travis-ci.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL)
[![Go Report Card](https://goreportcard.com/badge/github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL)](https://goreportcard.com/report/github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL)
[![codecov](https://codecov.io/gh/DigitalChinaOpenSource/TiDB-for-PostgreSQL/branch/main/graph/badge.svg?token=OZ16DNE6JH)](https://codecov.io/gh/DigitalChinaOpenSource/TiDB-for-PostgreSQL)

## Introduction

TiDB for PostgreSQL is an open source launched by Digital China Cloud Base to promote and integrate into the open source community of TiDB. In addition to its high availability, horizontal scalability, and cloud-native features, TiDB for PostgreSQL is compatible with the PostgreSQL protocol. On the one hand, you can use the PostgreSQL client to connect to TiDB for PostgreSQL, which we have basically completed. On the other hand, you can also use the unique grammatical features of PostgreSQL in your business system, which is in the development process.

More details can be found in the [Design Document](https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL/blob/main/DESIGN_DOCUMENT.md)

## Background

As the general trend of the massive explosion of information in the current data era becomes more and more obvious, the database is more and more likely to become the bottleneck of the upper-level business system. For this problem, TiDB solves most of migration problems of business system based on MySQL database through its highly compatibility of MySQL protocol. In view of its distributed transactions with strong consistency, flexible scalability, and excellent disaster recovery backup architecture, users prefer to use mature solutions of TiDB rather than scale MySQL by sharding. However, there is no mature solution for the upper-level business system based on PostgreSQL to migrate to TiDB. If you want to migrate a PostgreSQL-based business system to TiDB, you have to modify lots of code of business system. To solve this，we try to refactor the underlying source code of TiDB to make it compatible with the PostgreSQL protocol, making it possible to migrate PostgreSQL-based business systems to TiDB without modifying much code.



## Development progress

The largest amount of work for the development of TiDB for PostgreSQL is compatibility with PostgreSQL, which has two things to do. One is to implement the PostgreSQL connection protocol, which we have basically achieved, So PostgreSQL client can connect to TiDB for PostgreSQL. The other is the unique PostgreSQL syntax. TiDB for PostgreSQL can support general sql syntax, but there are a certain amount of work to do to make it compatible with PostgreSQL’s unique syntax. This is one of the reasons why TiDB for PostgreSQL is open sourced. We hope that through the open source, engineers interested in our TiDB for PostgreSQL can work together to make our project bigger and better.



## Quick start

First, make sure you have a Go environment, because TiDB for PostgreSQL is based on the Go language.

TiDB for PostgreSQL can be started on a single node without pd and tikv.

If there is no pd and tikv, it will create mock pd and mock tikv to maintain the stable operation of the system.

The following is an example of locally compiling and running TiDB for PostgreSQL on localhost.

```shell
mkdir -p  $GOPATH/src/github.com/digitalchina

cd  $GOPATH/src/github.com/digitalchina

git clone https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL.git

cd TiDB-for-PostgreSQL/tidb-server

go run main.go

# If you get an error: export ordinal to larger
# Please run the following cmd:
go run -buildmode=exe  main.go
```

After starting the main program of TiDB for PostgreSQL , it will run on port 4000 of the host.

How to use PostgreSQL client to connect to TiDB for PostgreSQL? Here, we take the command line tool psql that comes with PostgreSQL as an example.

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

## Docker 

```shell
# Log in to your dockerhub account
docker login

# Pull the image and start the container
docker pull dcleeray/tidb-for-pg
docker run -it --name tidbforpg -p 4000:4000 -d  dcleeray/tidb-for-pg:latest
```

## Cluster deployment
TiDB for PostgreSQL also supports cluster deployment.

Our current modification work does not involve the communication module of each component in the cluster. Therefore, the connection of TiDB for PostgreSQL to pd and tikv will not be affected in any way.

We recommend using binary file to deploy TiDB for PostgreSQL cluster.

First, download the official binary package file and unzip it.

```shell
wget http://download.pingcap.org/tidb-v4.0.11-linux-amd64.tar.gz
wget http://download.pingcap.org/tidb-v4.0.11-linux-amd64.sha256

sha256sum -c tidb-v4.0.11-linux-amd64.sha256

tar -xzf tidb-v4.0.11-linux-amd64.tar.gz
cd tidb-v4.0.11-linux-amd64/bin
```

Second, deploy each node in the cluster in order. According to the cluster architecture of TiDB for PostgreSQL, pd nodes are deployed first, then the tikv node, and finally the TiDB for PostgreSQL node. The number of nodes in the cluster is not specified, but there is at least one of each type.

Deploy a pd node

```shell
./pd-server --name=pd1 --data-dir=pd1 --client-urls="http://pdhost:2379" --peer-urls="http://host:2380" -L "info" --log-file=pd.log
```

Deploy three tikv nodes

```shell
./tikv-server --pd="pdhost:2379" --addr="kvhost1:20160"  --data-dir=tikv  --log-file=tikv1.log
./tikv-server --pd="pdhost:2379" --addr="kvhost2:20160"  --data-dir=tikv  --log-file=tikv2.log
./tikv-server --pd="pdhost:2379" --addr="kvhost3:20160"  --data-dir=tikv  --log-file=tikv3.log
```

Deploy a TiDB for PostgreSQL node. You need to compile the TiDB for PostgreSQL project into a binary file named tidb-server and upload it to this server for replace the original tidb-server

```shell
./tidb-server --store=tikv  --path="pdhost:2379" --log-file=tidb.log
```

With the above done, a TiDB for PostgreSQL cluster is successfully deployed, and you can connect to TiDB for PostgreSQL cluster in the same way as you did with single TiDB for PostgreSQL demonstrated earlier.

## Contribute code to TiDB for postgresql

We greatly welcome and appreciate developers who are interested in TiDB for PostgreSQL to make contributions. At present, we provide several directions to get started.

[Learning Guide](https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL/blob/main/LEARNING_GUIDANCE.zh.md)

[Contribution Guide](https://github.com/DigitalChinaOpenSource/TiDB-for-PostgreSQL/blob/main/CONTRIBUTING.md)

### Schema structure

The database structure of PostgreSQL is very different from that of MySQL, which is also a big problem we encountered in our development. It involves the realization of database system tables, system views, and system functions. It’s a lot of work and very hard. Developers are required to be very familiar with the database table structure of both PostgreSQL and MySQL.

### Postgresql grammatical features

Compared with MySQL, PostgreSQL has many new syntax features. For example, Returning can return specified column or all columns of the modified row. Implementing a syntax involves modifying codes of the Parser module, TiDB for PostgreSQL's internal plan structure, plan optimization, plan execution, and data write-back, and many other parts. It’s really very hard. We recommend that developers start with clauses such as RETURNING and modify them on the basis of the original code. After being familiar with the logic of TiDB for PostgreSQL planning and execution, you can try to implement a brand new statement.

## Version notice

To keep our code improvement running stable, we currently choose a fixed version for development. The following is our selected version:

TiDB v4.0.11

PostgreSQL 13

## License

Apache-2.0 License
