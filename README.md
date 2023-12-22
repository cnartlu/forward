# forward
一个简单的tcp转发层，支持验证来源IP的组织或域，实现简单的防火墙

# 使用方法

-r []string 转发规则 格式loaclAddr/remoteAddr

-o []string 组织cloudflare google

-d []string IP所属域

-remote-db string 远程下载的文件地址，默认值 https://github.com/cnartlu/geoip2/releases/download/V2023121819/asn.mmdb

systemd.service 创建
```
cat << EOF > /etc/systemd/system/forward.service
[Unit]
Description=forward
After=systemd-user-sessions.service

[Service]
Type=simple
WorkingDirectory=/usr/bin
ExecStart=/usr/bin/forward -r :80/remote:80 -r :443/remote:443 -o ori
User=www-data
Group=www-data

[Install]
WantedBy=multi-user.target
EOF
```
# ip数据支援
<a href="https://ipinfo.io">IPinfo</a>

