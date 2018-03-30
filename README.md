[![Go Report Card](https://goreportcard.com/badge/github.com/cbeuw/GoQuiet)](https://goreportcard.com/report/github.com/cbeuw/GoQuiet)

[简体中文](https://github.com/cbeuw/GoQuiet/wiki/GoQuiet)
# GoQuiet
A shadowsocks plugin that obfuscates the traffic as normal HTTPS traffic and disguises the proxy server as a normal webserver.

The fundamental idea of obfuscating shadowsocks traffic as TLS traffic is not original. [simple-obfs](https://github.com/shadowsocks/simple-obfs) and ShadowsocksR's `tls1.2_ticket_auth` mode have shown this to be effective. This plugin has made [improvements](https://github.com/cbeuw/GoQuiet/wiki/Advantages-over-similar-obfuscators) so that the goal of this plugin is  **to make indiscriminate blocking of HTTPS servers (or even IP ranges) with high traffic the only effective way of stopping people from using shadowsocks.**

Beyond the benefit of bypassing the firewall, it can also **cheat traffic restrictions imposed by ISP. See [here](https://github.com/cbeuw/GoQuiet/wiki/A-potential-gateway-to-free-internet-after-Net-Neutrality-Repeal).**

This plugin has been tested on amd64 and arm Linux and amd64 Windows. It uses about the same CPU and memory as shadowsocks-libev (which is very little), and has almost no transmission overhead added on top of shadowsocks. 

## Download

**Download the binaries [here](https://github.com/cbeuw/GoQuiet/releases)**

## Build

**Build with go1.8 or 1.9 only: https://github.com/cbeuw/GoQuiet/issues/26**

`make client` or `make server`

## Usage

**Change the key in config file before using it. It can be the same as shadowsocks' password**

You can check [Instructions for Windows users](https://github.com/cbeuw/GoQuiet/wiki/Instructions-for-Windows-Client-Users)

### Plugin mode

For server:

`ss-server -c <path-to-ss-config> --plugin <path-to-gq-server-binary> --plugin-opts "<path-to-gqserver.json>"`

For client:

`ss-local -c <path-to-ss-config> --plugin <path-to-gq-client-binary> --plugin-opts "<path-to-gqclient.json>"`

or as value of `plugin` and `plugin_opts` in Shadowsocks JSON

```json
{
    "server":"0.0.0.0",
    "server_port":443,
    "local_address": "127.0.0.1",
    "local_port":1080,
    "password":"mypassword",
    "timeout":300,
    "method":"aes-128-gcm",
    "fast_open":true,
    "reuse_port":true,
    "no_delay":true,
    "plugin":"path-to-gqserver/client-binary",
    "plugin_opts":"path-to-gqserver/client.json"
}
```

### Standalone mode

Standalone mode should only be used if your shadowsocks port does not support plugins

For server:
```
gq-server -r 127.0.0.1:8388 -c <path-to-gqserver.json>
ss-server -c <path-to-ss-config> -s 127.0.0.1 -p 8388
```
For client:
```
gq-client -s <server_ip> -l 1984 -c <path-to-gqclient.json>
ss-local -c <path-to-ss-config> -s 127.0.0.1 -p 1984 -l 1080
```

### Configuration

For server:

`WebServerAddr` is the redirection address and port when the incoming traffic is not from shadowsocks. It be the IP record of the `ServerName` set in `gqclient.json`

`Key` is the key. This needs to be the same as the `Key` set in `gqclient.json`

`FastOpen` is used to enable or disable TCP fast open.

For client:

`ServerName` is the domain you want to make the GFW think you are visiting

`Key` is the key

`TicketTimeHint` is the time needed for a session ticket to expire and a new one to be generated. Leave it as the default.

`Browser` is the browser you want to **make the GFW _think_ you are using, it has NOTHING to do with the web browser or any web application you are using on your machine**. Currently support `chrome` and `firefox`.

`FastOpen` is used to enable or disable TCP fast open.

## How it works
As mentioned above, this plugin obfuscates shadowsocks' traffic as TLS traffic. This includes adding TLS Record Layer header to application data and simulating TLS handshake. Both of these are trivial to implement, but by manipulating data trasmitted in the handshake sequence, we can achieve some interesting things.

A TLS handshake sequence is initiated by the client sending a `ClientHello` message. We are interested in the field `random` and `extension:session_ticket`. Accroding to [rfc5246](https://tools.ietf.org/html/rfc5246), the `random` field is the current 32bit unix time concated with 28 random bytes. However, in most implementations all the 32 bytes are randomly generated (source: Wireshark). The `session_ticket` extension triggers a mechanism called session resumption, which allows the server to skip a lot of steps, most notably the `Certificate` message sent by the server. If you don't have a valid TLS certificate, you'll have to compose an invalid cert, which is a strong feature indicating that the server is a proxy. With the `session_ticket`'s presence, we don't need to give out this information.

The client side of this plugin composes the `ClientHello` message using this procedure:
```python
# Global variables
#   In config file:
preshared_key = '[A key shared out-of-band]'
ticket_time_hint = 3600 # In TLS implementations this is the time in seconds for a session ticket to expire. 
                        # Common values are 300,3600,7200 and 100800

#   Calculated on startup:
aes_key = sha256(preshared_key)
opaque = rand32int()

# Random:
iv = randbytes(16)
goal = sha256(str(floor(gettimestamp()/(12*60*60))) + preshared_key)
rest = aes_encrypt(iv,aes_key,goal[0:16])
random = iv + rest

# Session ticket
ticket = randbytes(192,seed=opaque+aes_key+floor(gettimestamp()/ticket_time_hint)))
```

Once the server receives the `ClientHello` message, it checks the `random` field. If it doesn't pass, the entire `ClientHello` is sent to the web server address set in the config file and the server then acts as a relay between the client and the web server. If it passes, the server then composes and sends `ServerHello`, `ChangeCipherSpec`, `Finished` together, and then client sends `ChangeCipherSpec`, `Finished` together. There are no useful informations in these messages. Then the server acts as a relay between the client and the shadowsocks server.

### Replay prevention
The `gettimestamp()/(12*60*60)` part is there to prevent replay:

The `random` field should be unique in each `ClientHello`. To check its uniqueness, the server caches the value of the `random` field. Obviously we cannot cache every `random` forever, we need to regularly clean the cache. If we set the cache expiration time to, say 12 hours, replay attemps within 12 hours will fail, but if the firewall saves the `ClientHello` and resend it 12 hours later, that message will pass the check on the server and our proxy is exposed. However, when `gettimestamp()/(12*60*60)` is in place, the replayed message will never pass the check because for replays within 12 hours, they fail to the cache; for replays after 12 hours, they fail to the uniqueness of the value of `gettimestamp()/(12*60*60)` for every 12 hours.

### Notes on the web server
If you want to run a functional web server on your proxy machine, you need it to have a domain and a valid certificate. As for the domain, you can either register one at some cost, or use a DDNS service like noip for free. The certificate can be obtained from [Let's Encrypt](https://letsencrypt.org/) for free. **The certificate is for your web server (e.g. Apache and Nginx) only. The GoQuiet plugin does not need a certificate.**

https://dcamero.azurewebsites.net/shadowsocks-goquiet.html - Detailed guide about "How to make your traffic look like simple tls traffic"

Or you can set the `WebServerAddr` field in the server config file as an external IP, and set the `ServerName` field in the client config file as the domain name of that ip. Because of the [Server Name Indication](https://en.wikipedia.org/wiki/Server_Name_Indication) extension in the `ClientHello` message, the firewall knows the domain name someone is trying to access. If the firewall sends a `ClientHello` message to our proxy server with an SNI we used, the destination IP specified in `WebServerAddr` will receive this `ClientHello` message and the web server on that machine will check the SNI entry against its configuration. If they don't match, the web server will refuse to connect and show an error message, which could expose the fact that our proxy machine is not running a normal TLS web server. If you match the external IP with its domain name (e.g. `204.79.197.200` to `www.bing.com`), our proxy server will become, effectively to the observer, a server owned by that domain.
