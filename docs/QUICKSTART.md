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

sha256sum -c tidb-latest-linux-amd64.sha256

tar -xzf tidb-latest-linux-amd64.tar.gz
cd tidb-latest-linux-amd64/bin
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
