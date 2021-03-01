
# Apache 软件发版流程

> author: wongoo@apache.org
> last updated: 2020-03-06

Apache开源软件是有社区驱动的，为了提高发布软件质量而指定了软件发布流程，本文主要介绍此流程，以给第一次发布打包的apacher参考。

如果你要准备打包一个apache软件了，想必你已经是一个项目的committer了，而且知道社区、PMC这些概念，而你现在还担任本次发布的 release manager 一职。

发版流程其实也很简单，无非如下：
1. 整理变更内容，打包并对打包文件签名；
2. 将签名文件上传apache svn仓库；
3. 发邮件请社区PMC大佬投票；
4. 投票通过后发一个投票结果通告邮件；
5. 发版
6. 发版邮件通告社区新版本发布；

下面详细整理发版的一些流程步骤，使用 dubbo 的子项目 dubbog-go-hessian2 发版为例！


## 1. 发版准备

发版文件需要签名，需要安装pgp工具.

```bash
$ brew install gpg
$ gpg --version
$ gpg --full-gen-key
	(1) RSA and RSA (default)  <-- RSA 类型
	What keysize do you want? (2048) 4096  <-- key大小为4096
	0 = key does not expire    <-- 永不过期
	Real name: Liu Yang
	Email address: wongoo@apache.org
	Comment: CODE SIGNING KEY

	gpg: /Users/gelnyang/.gnupg/trustdb.gpg: trustdb created
	gpg: key 7DB68550D366E4C0 marked as ultimately trusted
	gpg: revocation certificate stored as '/Users/gelnyang/.gnupg/openpgp-revocs.d/1376A2FF67E4C477573909BD7DB68550D366E4C0.rev'
	public and secret key created and signed.

	pub   rsa4096 2019-10-17 [SC]
	      1376A2FF67E4C477573909BD7DB68550D366E4C0
	uid                      Liu Yang (CODE SIGNING KEY) <wongoo@apache.org>
	sub   rsa4096 2019-10-17 [E]

$ gpg --list-keys	
	gpg: checking the trustdb
	gpg: marginals needed: 3  completes needed: 1  trust model: pgp
	gpg: depth: 0  valid:   1  signed:   0  trust: 0-, 0q, 0n, 0m, 0f, 1u
	/Users/gelnyang/.gnupg/pubring.kbx
	----------------------------------
	pub   rsa4096 2019-10-17 [SC]
	      1376A2FF67E4C477573909BD7DB68550D366E4C0
	uid           [ultimate] Liu Yang (CODE SIGNING KEY) <wongoo@apache.org>
	sub   rsa4096 2019-10-17 [E]


# 公钥服务器是网络上专门储存用户公钥的服务器
# 通过key id发送public key到keyserver
$ gpg --keyserver pgpkeys.mit.edu --send-key 1376A2FF67E4C477573909BD7DB68550D366E4C0
	gpg: sending key 7DB68550D366E4C0 to hkp://pgpkeys.mit.edu
# 其中，pgpkeys.mit.edu为随意挑选的keyserver，keyserver列表为：https://sks-keyservers.net/status/，为相互之间是自动同步的，选任意一个都可以。

# 如果有多个public key，设置默认key。修改 ~/.gnupg/gpg.conf
$ vi ~/.gnupg/gpg.conf
default-key 7DB68550D366E4C0

# 如果有多个public key, 也可以删除无用的key：
### 先删除私钥，再删除公钥
$ gpg --yes --delete-secret-keys shenglicao2@gmail.com   ###老的私钥，指明邮箱即可
$ gpg --delete-keys 1808C6444C781C0AEA0AAD4C4D6A8007D20DB8A4

## 由于公钥服务器没有检查机制，任何人都可以用你的名义上传公钥，所以没有办法保证服务器上的公钥的可靠性。
## 通常，你可以在网站上公布一个公钥指纹，让其他人核对下载到的公钥是否为真。
# fingerprint参数生成公钥指纹：
$ gpg --fingerprint wongoo

	pub   rsa4096 2019-10-17 [SC]
	      1376 A2FF 67E4 C477 5739  09BD 7DB6 8550 D366 E4C0
	uid           [ultimate] Liu Yang (CODE SIGNING KEY) <wongoo@apache.org>
	sub   rsa4096 2019-10-17 [E]
	# 将上面的 fingerprint （即 1376 A2FF 67E4 C477 5739  09BD 7DB6 8550 D366 E4C0）粘贴到自己的用户信息中：
	# https://id.apache.org  OpenPGP Public Key Primary Fingerprint:
```

> 详细参考：
> - 发布签名: http://www.apache.org/dev/release-signing.html
> - 发布策略: http://www.apache.org/dev/release-distribution
> - 将密钥上传到公共密钥服务器: https://www.apache.org/dev/openpgp.html#generate-key

## 2. 打包签名

准备打包前（尤其提第一次打包）需要注意以下内容:
- 每个文件的LICENSE头部是否正确, 包括 `*.java`, `*.go`, `*.xml`, `Makefile` 等
- LICENSE 文件是否存在
- NOTICE 文件是否存在
- CHANGE.md 是否存在 （变更内容格式符合规范）

以上可以参考其他已发布项目的配置。


```

# NOTICE: 这里切分支，分支名称不要和版本号（tag用）类似，不然会有冲突
$ git checkout -b 1.4

$ git tag -a v1.4.0-rc1 -m "v1.4.0 release candidate 1"

$ git push --tags

# 打包
$ git archive --format=tar 1.4 --prefix=dubbo-go-hessian2-v1.4.0/ | gzip > dubbo-go-hessian2-v1.4.0-src.tar.gz

# 签名
$ gpg -u wongoo@apache.org --armor --output dubbo-go-hessian2-v1.4.0-src.tar.gz.asc --detach-sign dubbo-go-hessian2-v1.4.0-src.tar.gz

# 验证签名
$ gpg --verify dubbo-go-hessian2-v1.4.0-src.tar.gz.asc dubbo-go-hessian2-v1.4.0-src.tar.gz

# hash
$ shasum -a 512 dubbo-go-hessian2-v1.4.0-src.tar.gz > dubbo-go-hessian2-v1.4.0-src.tar.gz.sha512

# 验证 hash
$ shasum --check dubbo-go-hessian2-v1.4.0-src.tar.gz.sha512

```

> 发布版本: http://www.apache.org/dev/release-publishing.html

## 3. 上传打包文件到svn仓库

```
$ svn checkout https://dist.apache.org/repos/dist/dev/dubbo

$ cd dubbo

# 更新
$ svn update

# 添加 签名 和 public key 到KEYS文件并提交到SVN仓库
# 这里是将公钥KEYS放到根目录, 有的项目放到本次打包文件目录
$ (gpg --list-sigs wongoo && gpg --armor --export wongoo) >> KEYS

$ mkdir -p dubbo-go-hessian2/v1.4.0-rc1

# 拷贝相关文件到新建目录下

$ tree dubbo-go-hessian2
dubbo-go-hessian2
└── v1.4.0-rc1
    ├── dubbo-go-hessian2-v1.4.0-src.tar.gz
    ├── dubbo-go-hessian2-v1.4.0-src.tar.gz.asc
    └── dubbo-go-hessian2-v1.4.0-src.tar.gz.sha512

$ svn add dubbo-go-hessian2
$ svn add dubbo-go-hessian2/*
$ svn status
$ svn commit  --username wongoo -m "Release dubbo-go-hessian2 v1.4.0-rc1"
```

> 详细参考: svn版本管理 https://www.apache.org/dev/version-control.html


## 4. 发投票 [VOTE] 邮件

发任何邮件都是有一定格式的，你加入社区邮件列表后，就会收到很多这样的邮件，多看看就知道了，具体邮件范本参考文章后面的邮件范本。

发完【VOTE】邮件，私下沟通群里面请大佬PMC投票。
PMC投票会对你上传打包文件进行相关检查,
详细可以了解孵化中的项目发布完整的检查项参考： https://cwiki.apache.org/confluence/display/INCUBATOR2/IncubatorReleaseChecklist

收到3个binding邮件且超过72小时后，就可以发 投票结果 [RESULT] [VOTE] 邮件了。

> 原则上只有PMC的投票才算binding邮件, 当然也可以由社区决定。

这一步骤最常见有以下问题：
- 文件签名有问题
- 引用项目LICENSE问题
- 单元测试不通过

> 另外需要注意: 一个apache项目可能包含很多子项目，项目的PMC可能只对主项目比较了解，	他们并不清楚如何将子项目跑起来，也不知道如何跑单元测试，最好在邮件中附带一个如何进行单元测试的连接。例如 PMC 最了解 java，但子项目是golang，python，js等，你需要告诉他们如何测试你的项目。

可以参考投票规则: https://www.apache.org/foundation/voting.html

## 5. 发布版本

当正式发布投票成功后，先发[Result]邮件，然后就准备 release package。
将之前在dev下发布的对应rc文件夹下的源码包、签名文件和hash文件拷贝到另一个目录 v1.4.0，
注意文件名字中不要rcxx (可以rename，但不要重新计算签名，hash可以重新计算，结果不会变)。

将release包移动到正式版目录。如果你的软件是需要客户从apache下载的，则这一步是必须的。如果不是，比如golang引用github打包地址的则可以忽略。
```
svn up
cd dubbo-go-hessian2
svn move v1.4.0-rc1 v1.4.0
svn status
svn commit  --username wongoo -m "Release dubbo-go-hessian2 v1.4.0"
```

移到发版目录后，还需要进行相应的正式版本发布， 这里将具体发布方式整理到单独的章节 `7. 不同语言版本发布`，因为发布流程马上就要结束了 ^v^


## 6. 新版本通告 ANNOUNCE 邮件

恭喜你你已经到发版最后一步了，邮件格式参考以下邮件范本！


## 7. 不同语言版本发布

### 7.1 golang

在 github 基于投票分支发布了 release 版本。

### 7.2 java

java项目发版需发布到java maven仓库。

TODO

### 7.3 js

js项目发版需发布到npm仓库。

TODO

### 7.4 python

TODO

## 8. 邮件范本

### 8.1. 提出发版投票

- TO: dev@dubbo.apache.org
- Title: [VOTE] Release Apache dubbo-go-hessian2 v1.4.0 RC1

```
Hello Dubbo/Dubbogo Community,

 This is a call for vote to release Apache dubbo-go-hessian2 version v1.4.0 RC1.

 The release candidates: https://dist.apache.org/repos/dist/dev/dubbo/dubbo-go-hessian2/v1.4.0-rc1/
 Git tag for the release: https://github.com/apache/dubbo-go-hessian2/tree/1.4
 Hash for the release tag: 4c31e88c35afe84c0321d9f12f036e6d3c8962d0
 Release Notes: https://github.com/apache/dubbo-go-hessian2/blob/1.4/CHANGE.md
 The artifacts have been signed with Key :7DB68550D366E4C0, which can be found in the keys file:
 https://dist.apache.org/repos/dist/dev/dubbo/KEYS

 The vote will be open for at least 72 hours or until necessary number of votes are reached.

 Please vote accordingly:
 [ ] +1 approve
 [ ] +0 no opinion
 [ ] -1 disapprove with the reason

 Thanks,
 The Apache Dubbo-go Team
 ```


### 8.2. PMC 投票邮件回复


范例1：
```
+1 approve   <-- 首先表明同不同意

I have checked:    <-- 其次要说明自己检查了哪些项

1.source code can build          <-- 能否构建
2.tests can pass in my local     <-- 单元测试能否通过
3. NOTICE LICENSE file exist     <-- 协议文件是否存在
4.git tag is correct             <-- git tag 是否正确

there is one minor thing that in change logs file, there is no space
between text And link. I suggest add one to make it looks better.  <--- 一些其他改进建议
```

范例2：
```
+1

I checked the following items:

[v] Are release files in correct location?                    <-- 发布文件目录是否正确
[v] Do release files have the word incubating in their name?
[v] Are the digital signature and hashes correct?             <-- 签名、hash是否正确
[v] Do LICENSE and NOTICE files exists?
[v] Is the LICENSE and NOTICE text correct?                   <-- 协议文本是否正确
[v] Is the NOTICE year correct?                               <-- 注意年份是否正确
[v] Un-included software dependencies are not mentioned in LICENSE or NOTICE?   <-- 没有包含协议或注意没有提到的软件依赖
[v] License information is not mentioned in NOTICE?           <-- 协议信息没有在注意中提及
[x] Is there any 3rd party code contained inside the release? If so:   <-- 是否包含第三方代码
     [ ] Does the software have a compatible license?
     [ ] Are all software licenses mentioned in LICENSE?
     [ ] Is the full text of the licenses (or pointers to it) in LICENSE?
     Is any of this code Apache licensed? Do they have NOTICE files? If so:
     [ ] Have relevant parts of those NOTICE files been added to this NOTICE file?
[v] Do all source files have ASF headers?                             <-- 是否所有源码都有ASF头部
[v] Do the contents of the release match with what's tagged in version control?   <-- 发布的文件是否和github中tag标记的版本一致
[x] Are there any unexpected binary files in the release?             <-- 是否包含不应该存在的二进制文件
[v] Can you compile from source? Are the instruction clear?           <-- 能否编译？指令是否明确？

On my mac laptop, I could compile successfully but there's one failed unit
test against net.go. I believe this issue [1] can be fixed with [2] in the
next release.       <-- 编译问题及建议

Is the issue minor?  <-- 编译存在的问题是否都是较小的？
[v] Yes [ ] No [ ] Unsure

Could it possibly be fixed in the next release?  <-- 能否在下一版本修复？
[v] Yes [ ] No [ ] Unsure

I vote with:   <-- 我的投票
[v] +1 release the software
[ ] +0 not sure if it should be released
[ ] -1 don’t release the software because...

Regards,
-Ian.

1. https://github.com/apache/dubbo-go/issues/207
2. https://github.com/apache/dubbo-go/pull/209
```

范例3：
```
+1

I checked the following items:

[√] Do LICENSE and NOTICE files exists?
[√] Is the LICENSE and NOTICE text correct?
[√] Is the NOTICE year correct?
[√] Do all source files have ASF headers?
[√] Do the contents of the release match with what's tagged in version control?
[√] Can you compile from source?
I could compile successfully but there's failed units test.  I run the unit
test refer to :https://github.com/apache/dubbo-go#running-unit-tests .
But I think it is not matter, the test can be fixed in next release.


I vote with:
[√] +1 release the software
```

范例4:
```
Great improvement over the previous release but there are still issues from the last vote that have not been resolved. e.g. [6][7][8]

Can someone tell me if these files [1][2][3][4][5] are just missing ASF headers or have a different license?

If they are just missing headers and [6][7][8] explained then it +1 form me, otherwise it’s probably a -1.

Can people please carefully check the contents, and write down what you checked, rather than just saying +1.

I checked:
- signatures and hashes good
- LICENSE is missing the appendix (not a major issue)
- LICENSE may be is missing some information[1][2][3][4][5]
- NOTICE is fine
- No binaries in source release
- Some files are missing ASF headers or other license headers [1][2][3][4][5] - please fix

Thanks,
Justin

1. dubbo-go-1.1.0/cluster/loadbalance/round_robin_test.go
2. dubbo-go-1.1.0/common/extension/router_factory.go
3. dubbo-go-1.1.0/config_center/configuration_parser.go
4. dubbo-go-1.1.0/config_center/configuration_parser_test.go
5. dubbo-go-1.1.0/registry/zookeeper/listener_test.go
6. dubbo-go-1.1.0/cluster/loadbalance/least_active.go
7. dubbo-go-1.1.0/protocol/RpcStatus.go
8. dubbo-go-1.1.0/filter/impl/active_filter.go
```


### 8.3. 发 [RESULT] [VOTE] 投票结果通知邮件

- TO: dev@dubbo.apache.org
- Title: [RESULT] [VOTE]: Release Apache dubbo-go-hessian2 v1.4.0 RC1


```
Hello Dubbo/Dubbogo Community,

The release dubbo-go-hessian2 v1.4.0 RC1 vote finished, We’ve received 3 +1 (binding) votes.

+1 binding, Stocks Alex
+1 binding, Ian Luo
+1 binding, Jun Liu

The vote and result thread:
https://lists.apache.org/thread.html/r8070f3b00984888069dd4ddad1bbc424cde51ea68b6ff0520e609e18%40%3Cdev.dubbo.apache.org%3E


The vote passed. Thanks all.
I will proceed with the formal release later.


Best regards,

The Apache Dubbogo Team
```


### 8.4. 发 Announce 发版邮件

- TO: dev@dubbo.apache.org
- [ANNOUNCE] Apache Dubbo version 2.7.4 Released

```
Hello Community,

The Apache Dubbo team is pleased to announce that the 2.7.4 has been
released.

Apache Dubbo™  is a high-performance, java based, open source
RPC framework. Dubbo offers three key functionalities, which include
interface based remote call, fault tolerance & load balancing, and
automatic service registration & discovery.

Both the source release[1] and the maven binary release[2] are available
now, you can also find the detailed release notes in here[3].


If you have any usage questions, or have problems when upgrading or find
any problems about enhancements included in this release, please don’t
hesitate to let us know by sending feedback to this mailing list or filing
an issue on GitHub[4].



[1] http://dubbo.apache.org/en-us/blog/download.html
[2] http://central.maven.org/maven2/org/apache/dubbo
[3] https://github.com/apache/dubbo/releases
[4] https://github.com/apache/dubbo/issues
```

## 9. 参考

- dubbo发布流程: http://dubbo.apache.org/zh-cn/docs/developers/committer-guide/release-guide_dev.html
- doris发布流程: https://github.com/apache/incubator-doris/blob/master/docs/documentation/cn/community/release-process.md
- spark发布流程: http://spark0apache0org.icopy.site/release-process.html


