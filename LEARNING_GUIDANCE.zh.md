## **TiDB for PostgreSQL 学习指南**

这个文档将会告诉你如何开始学习TiDB for PostgreSQL(PgSQL)源码，并让你可以尝试修改不同模块代码来实现你想要的功能。

### **TiDB for PostgreSQL**

TiDB for PgSQL是基于开源分布式数据库TiDB进行改造的数据库，主要是在TiDB中实现PgSQL的协议和功能，使TiDB可以兼容PgSQL数据库客户端和应用。

关于TiDB for PgSQL项目的来源可以参考该文档：

[TiDB for PostgreSQL—牛刀小试 - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/379181280)

### **入门学习**

#### 一. 学习数据库

在开始本项目之前，你需要弄懂下面几个问题：数据库是什么，什么是关系型数据库，什么是分布式数据库，什么是关系型分布式数据库。

这些你可以通过谷歌等搜索引擎进行学习，或者阅读相关书籍与文档。

当你对数据库有一定的了解和学习后，你就可以开始了解TiDB for PgSQL。

#### 二. 学习TiDB

由于TiDB for PgSQL是基于TiDB进行改造的，所以其的整体架构与代码思想依然还是TiDB原生，那么你想要深入的学习本项目，你可以从TiDB的学习开始。

当然不用担心TiDB上手困难，其实我们之前选择修改TiDB，而不是其他的开源分布式数据库，其中较为重要的一个原因就是TiDB的生态和社区非常强大，其官方有非常丰富的文档和教学视频供我们学习，让我们更快的掌握TiDB，并进行修改，遇到的问题也可以在社区中找到答案。所以你可以通过TiDB社区快速学习TiDB的架构和代码，其次，当你熟悉了TiDB之后，其实你也已经熟悉了本项目。

Github：[pingcap/tidb: TiDB is an open source distributed HTAP database compatible with the MySQL protocol (github.com)](https://github.com/pingcap/tidb)

官网：[TiDB Community](https://tidb.io/)

博客：[博客 | PingCAP](https://pingcap.com/blog-cn/)

#### 三. 学习PgSQL

本项目主要是想要实现TiDB对PgSQL的兼容，主要的开发任务就是在TiDB架构下，用Go语言实现PgSQL的功能。那么你需要开始学习PgSQL数据库，找到你想要实现的功能（文档和代码），了解其功能原理和实现过程中，在本项目中找到合适的位置进行实现。

PgSQL的学习可以通过其官方文档和社区来进行。

官网：[PostgreSQL: The world's most advanced open source database](https://www.postgresql.org/)

### **TiDB for PgSQL的实现**

本项目是基于TiDB进行修改的，目前为止，主要修改的代码涉及两个部分：

#### 一. 修改TiDB协议，使之兼容PgSQL协议。

TiDB 分为三层架构，TiDB Server（SQL执行引擎），TiKV Server（数据存储） 和 PD Server（中心调度）, 本文所讲到的TiDB主要是TiDB Server， SQL执行引擎，其余的TiKV 和 PD 不做涉及。

主要修改的部分也是TiDB Server 执行引擎这一层的代码，我们可以大致将执行引擎这一层的操作分为一下几个部分：

接受请求 -> SQL解析 -> 计划制定 -> 计划优化 -> 计划执行 -> 返回请求

第一步接受请求和最后一步返回请求，都是协议层需要做的事，TiDB自身是MySQL协议，可以与MySQL客户端和应用程序直接通信，但是PgSQL通信协议不同与MySQL，所以本项目第一件事就是将TiDB的协议层进行了改造，实现了PgSQL通信的协议，使普通的PgSQL客户端可以连接上TiDB，例如[psql ](https://tomcam.github.io/postgres/)。

当然为了兼容PgSQL协议，还涉及到了SQL解析和计划制定这部分的修改，所以目前实现的PgSQL协议总共涉及到协议层，解析层和计划层这三个层面的代码修改。

实现可以参考文档：
[TiDB源码阅读：用 PgSQL 客户端连接到 TiDB - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/379966342)

#### 二. 添加PgSQL系统表和函数。

为了兼容一些图形化客户端，我们在TiDB中创建了部分PgSQL的系统表和系统函数，因为很多图形化客户端，例如PgAdmin和Navicat，在创建正常的连接之后都会使用这些系统表和函数来查询相关的数据库信息。

在图形化客户端这一块的兼容现在并没有做的非常完善，PgAdmin和Navicat都可以正常连上，但部分功能不可使用，因为PgSQL系统结构与MySQL相差较大，其架构为 Database—Schema—Table，而MySQL为Database(Schema)—Table，想在TiDB中将Database与Schema两个概念分开还比较困难，目前还在努力中，当然我们也非常希望有贡献者来帮忙实现。

所以现在TiDB for PgSQL与TiDB 最大的区别在于通信协议不同，并且增加了额外的系统表和函数。

### **修改源码**

首先你需要下载源码，并安装Go环境和Go编译器，本地进行编译，这个在Readme 中都能找到，有了这一系列的操作后，你可以正式启动自己的本地项目。

在目录 tidb-server/main.go 文件中找到启动函数main()，可以使用调试模式启动，这样在代码的运行过程中，我们可以通过断点来了解代码逻辑和参数详情。

项目运行起来之后，使用psql连接上数据库，进行一些命令交互，同时运用断点来查看SQL语句处理的流程和经过的方法。只有熟悉了SQL执行过程后，才可以对其进行修改，添加功能。

那么如何在TiDB for PgSQL 中实现一个PgSQL的关键词或者语法。

1. 这里需要用到另一个仓库[DCParser ](https://github.com/DigitalChinaOpenSource/DCParser)，基于TiDB [Parser](https://github.com/DigitalChinaOpenSource/DCParser)开发。该仓库用于解析SQL语句，将文本语句解析成相应的树结构的结构体，所以想要实现一个关键字或者语法，首先需要在DCParser中添加该关键字和语法的解析模式。

2. 对于兼容的关键字和语法，根据其功能的不同，在计划制定，优化，执行三个阶段，对该关键字的功能进行实现。由于不同的关键字功能不一样，整个过程中可能需要在不同的位置去实现，所以这一块需要对TiDB的整个执行框架非常了解，大家可以通过TiDB社区进行学习。

3. 最后执行完成SQL语句后，由于MySQL和PgSQL的返回数据形式是不一样的，如果只是普通的关键字兼容，那么你可以不用操作最后一步，但是如果是一些特殊的语句进行返回，例如PgSQL的一些系统函数，需要返回特定的值，那么你需要最后修改返回的数据内容，这一块内容主要在通信协议中修改。

### **代码检验**

对于刚入门的人来说，要想实现一些功能和方法，难度非常大，数据库系统不同于普通系统，内部耦合非常大，同时数据库中约束也非常的多，所以在实现功能前，一定要熟悉自己修改的模块和代码功能，不能因为自己的改动，造成系统出现其他的问题。

所以我们需要保证自己实现代码的质量和正确性：

 - 在修改代码时需要添加足够且正确的代码注释。
   
 - 在功能实现时，尽量简洁自己的代码，同时保证单一职责。
   
 - 完成每一个功能后，都需要有相应的单元测试。
   
 - 功能实现完成后，保证所有的单元测试（除自己的还有其他人的）都能正常通过。

完成这一切后，你就可以提交自己的代码到主分支了。

当然你在提交之后，我们还会进行一系列的测试，包括单元测试，性能测试和混沌网格测试等等，来保证整个项目的质量。如果你担心自己的功能会对数据库性能和稳定性造成影响，可以在本地自行进行一系列测试。

### **未来功能**

这个项目的目标是在TiDB上兼容PgSQL，所以在未来会尝试不断的实现PgSQL的各种功能，就目前而言，在第一个版本发布前，我们还有以下一些功能需求大家的力量一起完成：

 - 完善PgSQL的通信协议，虽然PgSQL协议已经实现了，但是还存在部分的BUG和缺陷，需要修复和完善。  

 - 系统架构，将Database和Schema概念分离，兼容PosgreSQL的系统架构。

 - 完善系统库和系统参数，PgSQL中的系统库和参数仍有部分未实现。

 - 实现PgSQL关键字和语法，任意选择和实现。

 - 实现一些关键数据类型，例如 OID。
 
 - 逐渐升级到 TiDB 5.0。

这个项目是一个非常有意义的项目，很多人都期待着TiDB for PgSQL，所以我们十分欢迎每一个参与进来的人，感谢每一个贡献的人。
