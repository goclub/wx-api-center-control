# wx-api-center-control

微信中控服务器

## 功能

1. 自动维护 [access token](https://developers.weixin.qq.com/doc/offiaccount/Basic_Information/Get_access_token.html),且在并发时候不会出现多个access token.

## 运行

```shell
git clone https://github.com/goclub/wx-api-center-control
# 参考 config.example.yaml 的内容创建一个 config.yaml 文件
# 构建
go build -o main *.go
# 运行
./main
```


## 调用

```shell
curl --request POST \
  --url http://127.0.0.1:8254/wx-api-center-control/cgi-bin/token \
  --header 'Content-Type: application/json' \
  --header 'cache-control: no-cache' \
  --data '{\n	"appid": "在config.yaml中配置的appid",\n	"sk": "在config.yaml中配置的sk"\n}'
```