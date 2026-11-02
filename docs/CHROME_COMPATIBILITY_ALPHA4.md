# Chrome REALITY compatibility in alpha4

The 1.12 Neko core removed X25519MLKEM768 from REALITY ClientHellos even when
the profile said `chrome`. Keeping that removal globally breaks current Xray
servers, which require the hybrid share. Keeping only the modern hello regressed
the owner's existing profile: physical Android VPN timed out, while Edge worked.
The identical profile field therefore did not imply identical wire behavior.

On TECNO KL7 / Android 14 the failing path establishes TCP, then modern Chrome
bytes remain unacknowledged or no TLS response arrives. A single isolated request
can pass, but eight concurrent HTTPS requests failed 0/8; Edge passed 8/8 with
ordinary all-app routing. Changing MTU, TUN stack and socket binding did not fix
it. A handshake mutex also failed and was withdrawn. The particular network
component dropping packets has not been identified; DPI is not a proven cause.

## Runtime policy

For raw VLESS/REALITY with `chrome` or the default fingerprint, configuration
migration enables `reality.legacy_chrome_fallback`. The stored profile, link,
fingerprint, UUID, SNI and server settings are not rewritten. Other transports
and fingerprints keep their existing behavior. An explicit `false` is preserved
for configurations requiring only modern Chrome.

The modern hybrid hello is attempted first. If it fails or has not authenticated
after one second, a second TCP socket attempts the old Neko Chrome hello. The
modern attempt remains alive, so slow modern-only servers keep their original
time budget. Both attempts use the same REALITY authentication and verify the
server's certificate MAC; neither ordinary TLS nor TCP success is sufficient.
The losing socket is closed. Application data is sent only over the winner.

A successful legacy authentication is remembered only in this outbound's memory,
including its clones. Subsequent connections use that variant directly. No global
fingerprint replacement or persistent endpoint database is introduced. Restarting
the VPN recreates discovery; a rejected cached legacy variant also retries modern.
Legacy mode uses classical X25519, matching the old client; it does not guarantee
post-quantum key exchange. Strict users can disable the runtime compatibility flag.

## Validation

Controlled local Xray tests cover modern hybrid authentication and a path that
silently drops hybrid ClientHellos while forwarding legacy Chrome. Both transfer
110592 bytes over REALITY/Vision HTTPS and pass eight fresh parallel connections.
The classic-path fixture sees exactly one modern probe before the authenticated
legacy mode is learned. Explicit fallback opt-out rejects this fixture. A wrong
REALITY public key is rejected twice without learning a legacy mode.

The first physical candidate passes the saved `chrome` profile on TECNO with
ordinary app routing: built-in check PASS, eight independent HTTPS requests 8/8
through Android VPN with NL exit.

Final alpha4/version250 was installed over the existing app with the same signing
certificate. Installed APK SHA matched the artifact. Saved fingerprint remains
`chrome`; comparison with the previous temporary Edge export changes only `fp`.
The genuine VLESS parser plus default initialization reparses the link (including
the provider's hybrid-support parameter) and passes an authenticated URL test at
65ms without writing this temporary parse to the database. Full VPN with default
logging: domain DNS+HTTPS200/NL689ms, eight parallel requests8/8 in799ms.
Stopping/starting the VPN again: DNS+HTTPS200/NL690ms; app checks still pass.
MTU9000, gVisor0 and ordinary all-app routing are restored. Temporary diagnostics,
test package and private exports were removed; the VPN is left running.

APK `NekoBox-SB-1.4.3-sb1.14.4-alpha4-chrome-arm64-v8a.apk`, 25698621 bytes.
SHA-256: `3296c686084e98105f1108838db2a9aa3e0636de67b4cdea4a63d5cead0ad6a7`.
Library matches the AAR; signature, 16KiB alignment, assets, package/version and
absence of the owner's UUID in source/APK passed. Main remains unchanged.

Version: `1.4.3-sb1.14.4-alpha4-chrome`; runtime `1.14.4-neko.3`.
The full remaining protocol/lifecycle acceptance and Xray/XHTTP backend remain
separate work; this change addresses Chrome REALITY import compatibility.
