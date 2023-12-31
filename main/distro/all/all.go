package all

import (
	// The following are necessary as they register handlers in their init functions.

	_ "github.com/xmplusdev/xray-core/app/proxyman/inbound"
	_ "github.com/xmplusdev/xray-core/app/proxyman/outbound"

	// Required features. Can't remove unless there is replacements.
	// _ "github.com/xmplusdev/xray-core/app/dispatcher"
	_ "github.com/XMPlusDev/XMPlus-NoRelay/app/xdispatcher"

	// Default commander and all its services. This is an optional feature.
	_ "github.com/xmplusdev/xray-core/app/commander"
	_ "github.com/xmplusdev/xray-core/app/log/command"
	_ "github.com/xmplusdev/xray-core/app/proxyman/command"
	_ "github.com/xmplusdev/xray-core/app/stats/command"

	// Other optional features.
	_ "github.com/xmplusdev/xray-core/app/dns"
	_ "github.com/xmplusdev/xray-core/app/log"
	_ "github.com/xmplusdev/xray-core/app/metrics"
	_ "github.com/xmplusdev/xray-core/app/policy"
	_ "github.com/xmplusdev/xray-core/app/reverse"
	_ "github.com/xmplusdev/xray-core/app/router"
	_ "github.com/xmplusdev/xray-core/app/stats"

	// Inbound and outbound proxies.
	_ "github.com/xmplusdev/xray-core/proxy/blackhole"
	_ "github.com/xmplusdev/xray-core/proxy/dns"
	_ "github.com/xmplusdev/xray-core/proxy/dokodemo"
	_ "github.com/xmplusdev/xray-core/proxy/freedom"
	_ "github.com/xmplusdev/xray-core/proxy/http"
	_ "github.com/xmplusdev/xray-core/proxy/shadowsocks"
	_ "github.com/xmplusdev/xray-core/proxy/socks"
	_ "github.com/xmplusdev/xray-core/proxy/trojan"
	_ "github.com/xmplusdev/xray-core/proxy/vless/inbound"
	_ "github.com/xmplusdev/xray-core/proxy/vless/outbound"
	_ "github.com/xmplusdev/xray-core/proxy/vmess/inbound"
	_ "github.com/xmplusdev/xray-core/proxy/vmess/outbound"

	// Transports
	_ "github.com/xmplusdev/xray-core/transport/internet/domainsocket"
	_ "github.com/xmplusdev/xray-core/transport/internet/http"
	_ "github.com/xmplusdev/xray-core/transport/internet/kcp"
	_ "github.com/xmplusdev/xray-core/transport/internet/quic"
	_ "github.com/xmplusdev/xray-core/transport/internet/tcp"
	_ "github.com/xmplusdev/xray-core/transport/internet/tls"
	_ "github.com/xmplusdev/xray-core/transport/internet/udp"
	_ "github.com/xmplusdev/xray-core/transport/internet/websocket"
	_ "github.com/xmplusdev/xray-core/transport/internet/reality"

	// Transport headers
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/http"
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/noop"
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/srtp"
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/tls"
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/utp"
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/wechat"
	_ "github.com/xmplusdev/xray-core/transport/internet/headers/wireguard"

	// JSON & TOML & YAML
	_ "github.com/xmplusdev/xray-core/main/json"
	_ "github.com/xmplusdev/xray-core/main/toml"
	_ "github.com/xmplusdev/xray-core/main/yaml"

	// Load config from file or http(s)
	_ "github.com/xmplusdev/xray-core/main/confloader/external"

	// Commands
	_ "github.com/xmplusdev/xray-core/main/commands/all"
)
