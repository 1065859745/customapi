# 简介
公司原来每个程序里都用到一个java类来发送短信，内部共享和使用很不方便。
于是做了一个http的接口，通过http请求来执行发送短信的程序
# 程序参数说明
- -p 启动程序的端口，默认8018
- -P 请求时需要的验证密钥，默认为空
# 请求示例
```
curl --header "Authorization: key=xxxxx" "http://{{.Host}}/sendMsg?phones=1312xxxxxxx+15600xxxxxx+147939xxxxx&messages=Hello"
```
- 若以无验证密钥的启动方式启动可以省去 <kbd>--header</kbd>参数
- 其他形式的请求可以将<kbd>phones</kbd>参数中的<kbd>+</kbd>号替换成空格
