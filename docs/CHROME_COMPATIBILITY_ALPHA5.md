# REALITY compatibility: alpha5

Alpha5 adds an automatic compact TLS 1.3 Hello for raw VLESS/REALITY profiles
using Chrome. Existing links import with their original fingerprint/settings;
no manual fingerprint change or server rekey is required.

The compact variant retains X25519MLKEM768 (1216-byte share), the separate
X25519 share, REALITY authentication and certificate MAC verification. It omits
optional browser extensions and TLS 1.2 offerings. It is a compatibility Hello,
**not an exact Chrome 133 fingerprint**. Its TLS record is 1438 bytes with a
22-character SNI; longer SNI or custom ALPN changes this size.

Discovery tries compact modern first. Unmodified browser Chrome and legacy
Chrome remain staggered fallbacks (500 ms between attempts); the total original
handshake deadline is preserved. Only a successfully authenticated variant is
remembered, in memory for that outbound. Losing sockets are cancelled and closed.
An explicit `tls.reality.legacy_chrome_fallback=false` continues to disable this
automatic policy and sends the configured browser fingerprint unchanged.

## Evidence and limits

On 2026-11-03 the physical TECNO KL7/Android 14 and a separate official Windows
Xray 26.10.31 reproduced intermittent failure with ordinary modern Chrome.
Synchronized server metadata showed missing ClientHello tails or server replies
without advancing client ACKs. Ordinary HTTPS to the same listener/SNI passed
8/8, and one official Xray connection transferred 1 MiB. Guest nginx/queue/FD
limits did not explain the failure. The specific external filtering device or
provider remains unidentified.

In the same running Android VPN, changing only the diagnostic Hello variant
produced compact 8/8 → ordinary Chrome 0/8 → compact 8/8. DNS, HTTPS with NL exit
and a 1 MiB download passed with compact. Neither the profile nor the server was
changed. This establishes dependence on Hello shape/size, not which individual
removed extension is decisive.

Local fixtures cover current and previous Xray, HTTPS upload/download, eight
parallel connections, compact-only and browser-only paths, classic-only path,
incorrect public key rejection, explicit opt-out and serialized share layout.
The final alpha5 artifact (version255/core1.14.4-neko.5, no diagnostic switches)
was installed and its SHA-256 matched the build artifact. The saved profile URI
was identical before/after; Chrome, empty custom outbound, log0, MTU9000,
gVisor and ordinary all-app routing were preserved. Raw-link reimport passed;
full VPN passed DNS/HTTPS/NL, eight parallel requests and a 1 MiB download,
restart, cellular transport and return to Wi-Fi.

During prolonged failed-connection testing nginx subsequently exhausted its
separate 256-worker-connection limit and produced fast broken-pipe failures.
The same unchanged final APK passed after connections naturally drained;
those capacity failures must not be conflated with the earlier Hello-dependent
failure. Server capacity remediation and graceful-reload acceptance are tracked
in the server's own operational runbook, outside this app patch.

No blanket claim is made for all NekoBox features, Android devices or networks.
