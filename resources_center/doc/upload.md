## 上传文件

通过封装oss上传功能，简化上传流程和提高上传速度。可用于前端直接上传文件到oss和后端直接上传文件到oss

### 前端上传资源

前端想要上传一些资源，适用场景是用户上传图片、视频素材，op管理后台上传模版资源、客服回复图片资源等。

将整个上传过程分为3个实体。前端、业务后端、资源中心。
![流程图](doc/img/flow.jpg)

上图是完成一次上传的时序图，下面按照步骤一一解释：
1. 前端根据算法（[算法详情](#qetagfunc)）计算文件的hash值，称为qetag，业务后端根据qetag去资源中心（[接口](http://yapi.xiaoniangao.cn/project/300/interface/api/7159)）判断文件是否存在，如果存在整个上传结束，不存在继续
2. 此时资源中心不存在该文件，由业务后端判断此时前端上传的是什么素材（图片、音乐、视频），去请求资源中心（[接口](http://yapi.xiaoniangao.cn/project/300/interface/api/6984)），获取上传需要的信息。信息包括：上传url地址、上传所要填写的密钥验证信息、上传文件的id、上传的所需要填写的参数
3. 资源中心返回这些信息
4. 业务后端将资源中心返回的信息+业务后端自己的逻辑信息、逻辑字段返回给前端
5. 前端将这些信息填充好，根据信息中的上传url，直接携带文件向url发起上传请求，[请求示例](#upload)
6. 资源中将上传文件成功消息和资源详细信息（[消息格式](#mq_message)）放到mq中，由订阅mq的业务消费，[业务后端监听mq示例](#mq)
7. 前端显示上传成功的消息，至此一次上传流程完成

### 后端上传资源
1. 业务后端根据算法（[算法详情](#qetagfunc)）计算文件的hash值，称为qetag，业务后端根据qetag去资源中心（[接口](http://yapi.xiaoniangao.cn/project/300/interface/api/7159)）判断文件是否存在，如果存在整个上传结束，不存在继续
2. 此时资源中心不存在该文件，由业务后端根据上传的是什么素材（图片、音乐、视频），去请求资源中心（[接口](http://yapi.xiaoniangao.cn/project/300/interface/api/6984)），获取上传需要的信息。信息包括：上传url地址、上传所要填写的密钥验证信息、上传文件的id、上传的所需要填写的参数
3. 资源中心返回这些信息
4. 业务后端将资源中心返回的信息+业务后端自己的逻辑信息、逻辑字段组合好，将这些信息填充好，根据信息中的上传url，直接携带文件向url发起上传请求，[请求示例](#upload)
5. 资源中将上传文件成功消息和资源详细信息（[消息格式](#mq_message)）放到mq中，由订阅mq的业务消费，[业务后端监听mq示例](#mq)
6. 后端显示上传成功的消息，至此一次上传流程完成

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
## 请求示例

资源中心已经返回了必要的上传所需要的信息，业务后端应该再次封装资源中心的接口[文档](http://yapi.xiaoniangao.cn/project/300/interface/api/6984)，产出自己含有自己业务逻辑、业务字段的接口提供给前端。

### 上传协议
目前资源中心仅支持http的post协议上传，文件上限为5GB，后续会增加上传协议。
#### HTTP POST 协议
上传方需要组装一个符合 HTML 文件上传表单规范（参考[RFC1867](http://www.ietf.org/rfc/rfc1867.txt)）的 HTTP 请求，并以 POST 方式向域名发起请求，即可将指定文件上传到服务端

##### 协议
POST

##### 语法
`Content-Type`为`multipart/form-data`格式

##### 上传接口
为资源中心获取上传信息接口的`host`字段

##### 参数
上传所需的参数，由资源中心提供，由业务后端自行封装提供接口给前端。

由3部分组成：

1. 资源中心返回的`upload_info`所有的字段
2. 自定义字段key为`x:my_var`
3. 为file字段，文件或文本内容，必须是表单中的最后一个域。浏览器会自动根据文件类型来设置Content-Type，并覆盖用户的设置。一次只能上传一个文件。

| 名称 | 类型 | 是否必须 | 描述 |
| :---- | :---- | :---- | :---- |
| xxx资源中心返回的 | 字符串 | 是 | 资源中心返回的`upload_info`所有的字段 |
| x:my_var | 字符串 | 是 | 因为是前端直接传文件，没有经过业务后端，一些自定义参数可以通过此字段在mq消息中传递到业务后端（如mid信息）。资源中心返回`upload_custom_info`字段，业务后端可继续在此字段中添加字段，然后将此字段的值通过base64 encode |
| file | 字符串 | 是 | 文件或文本内容，必须是表单中的最后一个域。浏览器会自动根据文件类型来设置Content-Type，并覆盖用户的设置。一次只能上传一个文件。 |

##### 返回
```json
{
    "ret": 1,
    "data": {
        "id": "3738536"
    }
}
```

<a name="mq"></a>
## mq 回调

<a name="mq_message"></a>
### mq 格式
消息编码为json字符串

| 字段名 | 类型 | 是否必传 | 描述 |
| :---- | :---- | :---- | :---- |
| id | 字符串 | 是 | 资源id |
| ty | 数字 | 是 | 资源类型，1图片，6视频，7音乐 |
| size | 数字 | 是 | 文件大小，单位B |
| qetag | 字符串 | 是 | 文件hash |
| upt | 数字 | 是 | 文件上传完成时间戳，单位毫秒 |
| src | 字符串 | 是 | 文件上传来源 |
| fmt | 字符串 | 是 | 文件格式 |
| w | 数字 | 否 | 文件宽，只有图片、视频会返回，单位px |
| h | 数字 | 否 | 文件高，只有图片、视频会返回，单位px |
| du | 数字 | 否 | 文件时长，只有音乐、视频会返回，单位毫秒 |
| cover | 字符串 | 否 | 封面资源ID，只有视频返回 |
| code | 字符串 | 否 | 视频编码，只有视频返回 |
| my_var | 字符串 | 是 | 业务自定义参数json字符串base64编码 | 

### mq 回调 topic
资源中心上传功能的消息通知以产品类型作为topic分类标准，命名方式为topic_upload_ + 产品类型缩写，如小年糕的服务需订阅 ”topic_upload_xng“这个topic，具体topic名称请参考下表。

| 产品 | topic |
| :---- | :---- |
| 小年糕 | topic_upload_xng |
| 小板凳 | topic_upload_xbd |
| TIA | topic_upload_tia |

### mq 回调 tag
topic内的资源会根据上传来源设置tag，业务端可以根据tag进行消息过滤，只接收自己需要的资源，tag的命名方式为 src_ + 上传来源编号，如微信小程序上传的资源tag为 src_11。

| 项目 | tag |
| :---- | :---- |
| 微信小程序 | src_11 |
| App | src_13 |
| OP | src_9 |
| 微信公众号命令行上传 | src_6 |

### 业务接受回调demo
[demo](doc/demo/main.go)