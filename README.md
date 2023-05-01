# LightSocks
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fxmapst%2Flightsocks.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fxmapst%2Flightsocks?ref=badge_shield)

support socks4, socks4a, socks5, socks5h, http proxy all in one

## Help

```bash
Support socks4, socks4a, socks5, socks5h, http proxy all in one,

Usage:
  ./bin/lightsocks_darwin_amd64 [flags]

Flags:
  -c, --config string   config file path (default "config.yaml")
  -h, --help            help for /lightsocks

```

## Dashboard

### Logs
![Logs](https://raw.githubusercontent.com/xmapst/lightsocks/main/img/logs.png)

### Connections
![Connections](https://raw.githubusercontent.com/xmapst/lightsocks/main/img/connections.png)

## Build

```bash
git clone https://github.com/xmapst/lightsocks.git
cd lightsocks
make
```

## Direct Mode

see `config.yaml` configure

```bash
./lightsocks -c example/config.yaml
```

## Proxy Mode

see `server.yaml` and `client.yaml` configure

```bash
sitsed ./lightsocks -c example/server.yaml 
sitsed ./lightsocks -c example/client.yaml
```

## Service

### Linux
```bash
echo > /etc/systemd/system/lightsocks.service <<EOF
[Unit]
Description=lightsocks - Support socks4, socks4a, socks5, socks5h, http proxy all in one
Documentation=https://github.com/xmapst/lightsocks
After=network.target nss-lookup.target

[Service]
NoNewPrivileges=true
ExecStart=/usr/local/bin/lightsocks -c /etc/lightsocks.yaml
Restart=on-failure
RestartSec=10s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now lightsocks.service
```

### Windows

```powershell
New-Service -Name lightsocks -BinaryPathName "C:\lightsocks\lightsocks_windows_amd64.exe -c C:\lightsocks\client.yaml" -DisplayName  "lightsocks " -StartupType Automatic
sc.exe failure lightsocks reset= 0 actions= restart/0/restart/0/restart/0
sc.exe start lightsocks
```

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fxmapst%2Flightsocks.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fxmapst%2Flightsocks?ref=badge_large)
