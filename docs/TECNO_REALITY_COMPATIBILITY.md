# TECNO KL7 / Android 14: совместимость текущего профиля

02.11.2026. ARM64 APK alpha3/version245, sing-box1.14.4-neko.2.

Физический телефон, тот же VLESS TCP/REALITY/Vision профиль:

- Chrome без TUN: отдельный test71ms, полная конфигурация74ms PASS.
- Chrome в полноценном VPN: повторяемый timeout3000ms.
- Edge в полноценном VPN: PASS75/66ms.
- Независимый UID: HTTPS по IP200/VPN=true/NL919ms; DNS+HTTPS по имени200/NL587ms.

Возврат Edge→Chrome воспроизвёл отказ; Chrome→Edge снова дал рабочий VPN.
Экспорт профиля до/после сравнен приватно: отличается только поле `fp`.
Ключ, сервер, SNI, short ID и Vision сохранены. SHA256 установленного APK совпал
с исходным alpha3. Кандидат socket bind не помог и отозван. MTU9000, gVisor и
logLevel0 восстановлены; тестовый APK, ADB forwards и приватные копии удалены.

Edge — настройка совместимости данного профиля, не глобальная подмена Chrome.
Он использует классический key share без X25519MLKEM768. Новые Xray-серверы,
требующие hybrid ClientHello, должны использовать поддерживаемый современный
fingerprint; нельзя менять все профили на Edge. REALITY authentication и TLS
verification не отключались. Причина отказа Chrome в полном VPN на этом устройстве
остаётся не установленной; DPI/ARM crypto не являются доказанной причиной.
Полная приёмка всех протоколов, Doze и маршрутизации остаётся впереди.
