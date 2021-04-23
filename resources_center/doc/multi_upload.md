## 分片上传

分片上传突破了原本资源中心上传功能的大小限制，结构模式不同于原本的post上传，采用了兼容多个平台的上传方式。

### 前端上传资源

前端想要上传一些资源，适用场景是用户上传视频素材。

将整个上传过程分为4个实体。前端、业务后端、资源中心、云存储平台。
![流程图](doc/img/multi_upload.png)

上图为实体间交互的流程，下面按照步骤一一解释：
1. 前端根据算法（[算法详情](#qetagfunc)）计算文件的hash值，称为qetag，业务后端根据qetag去调用资源中心的获取资源上传状态接口来判断文件是否存在，如果存在则结束上传流程
2. 资源中心未存储该资源（0）或资源状态为上传中断（3）：去请求资源中心获取上传配置信息。信息包括：分片策略、分片总数、资源名、上传事件ID（upload_id）和已上传分片信息（若无任何已上传分片，此字段为nil）
3. 资源中心返回上传配置
4. 业务后端将资源中心返回的信息返回给前端
5. 前端根据分片策略切分资源，循环的计算当前分片资源的md5值，使用md5值请求业务端获取分片授权信息，业务端调用分片授权接口获取授权信息
6. 前端使用授权信息进行分片上传，分片策略中包含ready字段，ready为1的分片代表已上传过，可以忽略该分片达到断点续传的效果，上传新分片后存储返回的etag值和分片编号，拼接到已有的上传分片信息之后（parts）
7. 前端全部上传完毕后，请求业务端进行上传结果的校验，业务端调用校验分片上传结果接口
8. 资源中心进行资源合并，成功则进行资源信息的获取，将结果和资源信息返回给业务端，业务端将信息返回给前端
9. 前端为用户显示上传结果


<a name="qetagfunc"></a>
## qetag算法

暂时使用的是七牛的hash算法

七牛的 hash 算法是公开的。见： [https://github.com/qiniu/qetag](https://github.com/qiniu/qetag)

算法大体如下：

如果你能够确认文件 <= 4M，那么 `hash = UrlsafeBase64([0x16, sha1(FileContent)])`

如果文件 > 4M，则 `hash = UrlsafeBase64([0x96, sha1([sha1(Block1), sha1(Block2), ...])])`

其中 Block 是把文件内容切分为 4M 为单位的一个个块，也就是 `BlockI = FileContent[I*4M:(I+1)*4M]`

[go语言版本](https://github.com/qiniu/qetag/blob/master/qetag.go)

[js语言版本](https://github.com/qiniu/qetag/blob/master/qetag.js)

<a name="upload"></a>
## 接口字段详情

相关接口内容请参考：https://xconfluence.xiaoniangao.cn/wiki/pages/viewpage.action?pageId=110501331

### 上传协议
目前资源中心分片上传功能支持48.8G以下大小的视频资源，上传分片采用http的PUT方式发起请求。
#### HTTP PUT 协议
上传方需要在请求header中设置Content-MD5（分片的md5值），Content-Type，Host，Content-Length（分片大小），Date（GMT格式时间），Authorization（授权签名），并添加x-oss-security-token（上传uptoken）

##### 协议
PUT

### 业务接受回调demo
[demo](doc/demo/multi_upload.go)