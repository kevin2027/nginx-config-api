# Nginx Config API

Nginx Config API 是一个用于动态管理 Nginx 配置的服务，可以通过 RESTful API 对 Nginx 配置进行上传、修改、删除和查询操作，并支持配置热加载。

## 特性

- 支持上传、修改、删除和查询 Nginx 配置文件
- 支持配置热加载
- 日志记录和切割功能

## 安装与运行

```bash
curl -fsSL https://raw.githubusercontent.com/kevin2027/nginx-config-api/main/install.sh | sudo bash
```

## API 文档

- **上传配置文件**

  ```bash
  POST /api/ngx/configs/{filename}
  ```

  请求体：

  ```json
  {
    "content": "配置文件内容"
  }
  ```

- **修改配置文件**

  ```bash
  PUT /api/ngx/configs/{filename}
  ```

  请求体：

  ```json
  {
    "content": "更新后的配置文件内容"
  }
  ```

- **删除配置文件**

  ```bash
  DELETE /api/ngx/configs/{filename}
  ```

- **查询配置文件列表**

  ```bash
  GET /api/ngx/configs
  ```

## 配置参数

- `NGX_CONF_API_CONFIG_DIR`：Nginx 配置文件存放目录，默认为 `/etc/nginx/conf.d/`
- `NGX_CONF_API_NGINX_PATH`：Nginx 可执行文件路径，默认为 `/usr/sbin/nginx`
- `NGX_CONF_API_LOG_DIR`：日志文件存放目录，默认为 `/var/log/nginx/agent`
- `NGX_CONF_API_HOST`：服务监听的主机地址，默认为 `0.0.0.0`
- `NGX_CONF_API_PORT`：服务监听的端口号，默认为 `5000`

可以通过设置以上环境变量来修改默认配置。

## 日志记录

日志文件存放在指定的日志目录中，并且会在日志文件达到一定大小时进行切割，保留一定数量的备份文件和一定的历史记录时间。

## 注意事项

- 请确保程序有足够的权限读取和写入 Nginx 配置文件以及写入日志文件。

## 维护者

- [YaoKevin](https://github.com/kevin2027)
