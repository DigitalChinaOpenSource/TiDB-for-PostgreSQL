DCTiDB的开发，需要引入我们自己修改过后的DCParser，而不是github.com/pingcap/parser，所以，这里需要变更引用。

步骤如下：

##### 1.首先配置go module为打开模式（这个其实无关紧要，但是开着更好）

`go env -w GO111MODULE=on`



##### 2.配置go module中的私有仓库地址

`go env -w GOPRIVATE="newgitlab.digitalchina.com"`



##### 3.因为拉取gitlab私有代码会默认http格式下载，所以需要更改一下拉取方式

`git config --global url."git@newgitlab.digitalchina.com:qiruia/dcparser.git".insteadOf "https://newgitlab.digitalchina.com/qiruia/dcparser.git"`



##### 4.Gitlab—>Settings—>Access Tokens，然后创建一个personal access token，这里权限最好选择只读(read_repository)。
注明：以上三步做完可以先测试一下，这一步不一定是必须的

`git config --global http.extraheader "PRIVATE-TOKEN: YOUR_PRIVATE_TOKEN"`



##### 5.在CMD中运行

`go get newgitlab.digitalchina.com/qiruia/dcparser`



拉取会报错，但是拉到本地的GOPATH中就行，不用理会。

报错原因是这个newgitlab.digitalchina.com/qiruia/dcparser 的真正名字是 github.com/pingcap/parser，dcparser那个只是个路径地址名字；

模块的名字是由每个项目中go.mod第一行的module决定的，这里不做修改，改了的话，包的名字就变了，那包里面的变量名称也就变了，对于别的项目来说，这个包中的变量名是包的name.variable，由于TiDB中依赖项太多，改不过来，就不给它更名了。

##### 6.在dctidb的go.mod 后面添加

`replace github.com/pingcap/parser v0.0.0-20210107054750-53e33b4018fe => newgitlab.digitalchina.com/qiruia/dcparser latest`

运行代码，会自动拉取最新的dcparser，但是有一点需要注意，代码运行，会将latest覆盖，修改为 v0.0.0-commitTime-commitHash这样的形势，所以在每一次dcparser更新的时候，我们都得重新修改go.mod，将latest添加上去，覆盖掉版本号，让它重新拉取。

最后说明一下为什么要将DCParser放在我自己的仓库：

由于基地这边代码管理是：wuhan/groupName/projectName，但是go module中不认，它只能识别到 wuhan/groupName，最后的projectName会被砍掉，最终会去执行 go get url/wuhan/groupName.git，但是那里面没有代码。所以，go get后面的地址得是：url/groupName/projectName，这样的一个形式。然后折中一下，先放在我的仓库中去了。
也许有解决办法，但我现在不知道怎么解决。

开发还是在 wuhan/DCTiDB/DCParser 这个项目中开发，只不过会在DCParser更新时，将这个仓库的代码拉下来，然后推到 qiruia/dcparser 这个仓库中。
