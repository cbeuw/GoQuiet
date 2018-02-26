[![Go Report Card](https://goreportcard.com/badge/github.com/cbeuw/GoQuiet)](https://goreportcard.com/report/github.com/cbeuw/GoQuiet)
# GoQuiet
A shadowsocks plugin obfuscates the traffic as normal HTTPS traffic and disguises the proxy server as a normal webserver.

ShadowsocksR's `tls1.2_ticket_auth` allows shadowsocks' traffic to be preceived by the firewall as normal TLS traffic, 
and [simple-obfs](https://github.com/shadowsocks/simple-obfs) allows the server to redirect non-shadowsocks traffic to a desired webserver, which defends against active detections from the firewall.

This plugin merges these two functionalities together, and also prevents the firewall from identifiying the nature of the proxy server through replaying shadowsocks' traffic.

Beyond the benefit of bypassing the firewall, it can also **deceive traffic restrictions imposed by ISP. See [this section](#a-potential-gateway-to-free-internet-after-net-neutrality-repeal)**

This plugin has been tested on amd64 and arm Linux and amd64 Windows. It uses about the same CPU and memory as shadowsocks-libev (which is very little), and has almost no transmission overhead added on top of shadowsocks. Of course this project is still **very early into development, stability is therefore not guareented.**

## Usage

**Change the password in config file before using it**

For server:
`go build .../GoQuiet/cmd/gq-server/`

`ss-server -c <path-to-ss-config> --plugin <path-to-gq-server-binary> --plugin-opts "<path-to-gqserver.json>"`

For client:
`go build .../GoQuiet/cmd/gq-client/`

`ss-local -c <path-to-ss-config> --plugin <path-to-gq-client-binary> --plugin-opts "<path-to-gqclient.json>"`

Or if you are using Shadowsocks client Windows GUI, put the `<path-to-gq-client.exe>` in Plugin field and `<path-to-gqclient.json>` in Plugin Options field

Compiled binaries will be released at some point.

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

`random` field should be unique in each `ClientHello`. To check its uniqueness, the server caches the value of the `random` field. Obviously we cannot cache every `random` forever, we need to regularly clean the cache. If we set the cache expiration time to, say 12 hours, replay attemps within 12 hours will fail, but if the firewall saves the `ClientHello` and resend it 12 hours later, that message will pass the check on the server and our proxy is exposed. However, when `gettimestamp()/(12*60*60)` is in place, the replayed message will never pass the check because for replays within 12 hours, they fail to the cache; for replays after 12 hours, they fail to the uniqueness of the value of `gettimestamp()/(12*60*60)` for every 12 hours.

### Notes on the web server
If you want to run a functional web server on your proxy machine, you need it to have a domain and a valid certificate. As for the domain, you can either register one at some cost, or use a DDNS service like noip for free. The certificate can be obtained from [Let's Encrypt](https://letsencrypt.org/) for free. (TODO: allow full TLS handshake using this cert)

Or you can set the `WebServerAddr` field in the server config file as an external IP, and set the `ServerDomainName` field in the client config file as the domain name of that ip. Because of the [Server Name Indication](https://en.wikipedia.org/wiki/Server_Name_Indication) extension in the `ClientHello` message, the firewall knows the domain name someone is trying to access. If the firewall sends a `ClientHello` message to our proxy server with an SNI we used, the destination IP specified in `WebServerAddr` will receive this `ClientHello` message and the web server on that machine will check the SNI entry against its configuration. If they don't match, the web server will refuse to connect and show an error message, which could expose the fact that our proxy machine is not running a normal TLS web server. If you match the external IP with its domain name (e.g. `204.79.197.200` to `www.bing.com`), our proxy server will become, effectively to the observer, a server owned by that domain.

## A potential gateway to free internet after Net Neutrality Repeal 
Given that American ISPs haven't yet started restricting internet traffic, we are not sure how they would implement it in the future. But some Chinese ISPs do selectively cause artificial packet loss based on the popularity of the protocol and/or its destination. For example, your HTTPS traffic to Baidu will be likely unaffected, but you will have a bad time trying to play Rainbow Six Siege. Should American ISPs go the same way, this plugin may help you to mitigate these restrictions:

You buy an f1-micro instance on Google Cloud Platform which costs $3.88 per month (or free if your usage is within the free tier limit), you set up shadowsocks and gqserver plguin on that machine, and you set `ServerDomainName` in your client config to a Google domain. If your broadband plan allows traffic to Google (which it would very likely do), you can deceive your ISP and pay no extra money to access the entirety of the Internet.

To reduce your GCP usage cost or even maintain it within the free tier limit, you can even use the Access Control List functionality provided by shadowsocks, which only proxies the traffic you want to be proxied (the websites you didn't pay extra money for).

This may be extremely useful if the ISPs turn out to use a whitelist to restrict access to the Internet (i.e. specify what is allowed and blocks everything else, hence you can't use VPN). Of course, these are just speculation. I don't want to see Americans using tools designed to mitigate Internet censorship to access a free Internet
