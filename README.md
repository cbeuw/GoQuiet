# [WIP]GoQuiet
A shadowsocks plugin that simulates TLS handshake and masquerades as a normal webserver.

ShadowsocksR's `tls1.2_ticket_auth` allows Shadowsocks' traffic to be preceived by the firewall as normal TLS traffic, 
and [simple-obfs](https://github.com/shadowsocks/simple-obfs) allows the server to redirect non-shadowsocks traffic to a desired webserver, which counters active detections from the firewall.

This plugin merges these two functionalities together, and also prevents the firewall from identifiying the nature of the proxyserver through replaying shadowsocks' traffic.

# How it works
As mentioned above, this plugin simulates TLS traffic. This includes adding TLS Record Layer header to packets, which is trivial, and simulating TLS handshake, through which we can do some interesting stuff.

The a TLS handshake sequence is initiated by the client sending a `ClientHello` message. We are interested in `random` and `extension:session_ticket` fields. Accroding to [rfc5246](https://tools.ietf.org/html/rfc5246), the `random` field is the current 32bit unix time concated with 28 random bytes. However, in most implementations all the 32 bytes are randomly generated. The `session_ticket` extension triggers a mechanism called session resumption, which allows the server to skip a lot of steps, most notably the `Certificate` message sent by the server. If you don't have a valid TLS certificate, you'll have to compose an invalid cert, which is a strong feature indicating that the server is a proxy. With the `session_ticket`'s presence, we don't need to give out this information.

The client side of this plugin would compose and send the `ClientHello` message, using this procedure:
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
goal = sha256(str(floor(gettimestamp()/ticket_time_hint)) + preshared_key)
iv = randbytes(16)
rest = aes_encrypt(iv,aes_key,goal[0:16])
random = iv + rest

# Session ticket
ticket = randbytes(192,seed=opaque+byte(preshared_key).to_int('big')+floor(gettimestamp()/ticket_time_hint)))
```

Once the server receives the `ClientHello` message, it checks the `random` field. If it doesn't pass, the entire `ClientHello` is sent to the web server address set in the config file and the server then acts as a relay between the client and the web server. If it passes, the server then composes and sends `ServerHello`, `ChangeCipherSpec`, `Finished` together, and then client sends `ChangeCipherSpec`, `Finished` together. There are no useful informations in these messages. Then the server acts as a relay between the client and the shadowsocks server.

## Notes on the web server
If you want the web server on your proxy machine to be functional, you need it to have a domain and a valid certificate. As for the domain, you can either register one at some cost, or use a DDNS service like noip for free. The certificate can be obtained from [Let's Encrypt](https://letsencrypt.org/) for free. (TODO: allow full TLS handshake using this cert)

Or you can set the web server field in the config file as some external address.

In the `ClientHello` message , there is a field called [Server Name Indication](https://en.wikipedia.org/wiki/Server_Name_Indication), which gives the server's domain name in plaintext. This needs to be changed in the Config file to your own domain name if you are running the web server on the proxy machine, or the external website's domain name. The default is s3-us-west-2.amazonaws.com.
