# 简介
公司原来每个程序里都用到一个java类来发送短信，内部共享和使用很不方便。
于是做了一个http的接口，通过http请求来执行发送短信的程序
# 程序参数列表
- -p 启动程序的端口，默认8018
- --config.file 配置文件目录，默认当前目录下的customapi.json
# 配置文件参数列表
|参数名称|是否必须|参数类型|说明|默认值|
|---|---|---|---|---|
|path|必须|string|请求路径||
|method|可选|string|请求方法|GET|
|parameters|可选|\[\]parameter|请求参数||
|commands|必须|\[\]string|执行命令||
|stdinPipe|可选|string|执行命令后向管道里传递的值||a
|output|可选|boolean|是否输出命令执行结果|false|
|pwd|可选|string|api的验证密钥||
## parameters
|参数名称|是否必须|参数类型|说明|默认值|
|---|---|---|---|---|
|name|必须|string|请求参数名称||
|require|可选|boolean|参数是否必须|false|
|pattern|可选|string|参数的值需满足的正则表达式||
|tip|可选|string|参数的提示信息||
