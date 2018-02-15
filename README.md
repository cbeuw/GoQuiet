# [WIP]GoQuiet
A shadowsocks plugin that simulates TLS handshake and masquerades as a normal webserver.

ShadowsocksR's `tls1.2_ticket_auth` allows Shadowsocks' traffic to be preceived by the firewall as normal TLS traffic, 
and [simple-obfs](https://github.com/shadowsocks/simple-obfs) allows the server to redirect non-shadowsocks traffic to a desired webserver,
which counters active detection from the firewall.

This plugin merges these two functionalities together, and also prevents the firewall from identifiying the nature of the proxyserver through replaying shadowsocks' traffic.
