通过ssh代理转发功能，连接服务器，内网服务


配置文件：

config.json
```
{
    "Host": "ssh ip",
    "Port": 22,
    "User": "ssh user",
    "Password": "ssh passowrd (if privateKey emtpy)",
    "PrivateKey": "rsa key file path",
    "Forward": [
        {
            "//": "example: forward mysql"
            "Enable": true,
            "LocalIP": "127.0.0.2",
            "LocalPort": 3306,
            "RemoteIP": "127.0.0.1",
            "RemotePort": 3306
        }
    ]
}
```
