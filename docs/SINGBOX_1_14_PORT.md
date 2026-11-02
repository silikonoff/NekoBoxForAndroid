# NekoBox SB: sing-box 1.14.4

02.11.2026. Этап 1: сохранение интерфейса и функций NekoBox при обновлении ядра.
Xray/XHTTP в Android-клиенте — следующий этап. Это alpha, не завершённая приёмка
всех функций на устройстве. TECNO KL7/Android14 проверен непосредственно по USB;
Chrome REALITY acceptance: [alpha4 compatibility](CHROME_COMPATIBILITY_ALPHA4.md).

## Исходники и версии

| Компонент | Закреплённый источник |
|---|---|
| NekoBox baseline | `5768494d8ae3c74a057bb6d46c0f8dc071b0d821`, upstream1.4.2 |
| sing-box | `c005f35300ce76427dc285481ec5695f5b179cd2`, upstream v1.14.4 |
| libneko | `1c47a3af71990a7b2192e03292b4d246c308ef0b` |
| gomobile | `17d6af34f6bd6d7e1e428e0c652c8b54a46bda4f`, MatsuriDayo fork |
| Проверенная локальная сборка | Go1.25.5, JDK21, SDK35, build-tools35.0.1, NDK28.2.13676358 |
| Runtime label | `1.14.4-neko.3` — не немодифицированное upstream ядро |
| Android package | `io.github.silikonoff.nekobox`, debug suffix `.debug` |

`buildScript/lib/core/get_source_env.sh` фиксирует коммиты зависимостей.
`buildScript/patches/nekobox-core.patch` применяется поверх чистого upstream1.14.4.
Необходимые дополнения исходного Neko сохранены отдельным патчем, а не скрытой
зависимостью от плавающей ветки ядра. GPL notices и оригинальные AUTHORS сохранены.

## Что переносится

- Kotlin/Java интерфейс, БД, импорт/экспорт, группы, подписки, цепочки и редакторы
  протоколов сохраняются. Перезаписи пользовательской БД при переносе не требуется.
- Слой `libcore/configcompat` преобразует прежний JSON только при запуске ядра.
  Сохранённый профиль и текст подписки остаются исходными. Неизвестные поля
  передаются настоящему парсеру ядра, который выдаёт ошибку, если они не поддержаны.
- Прежний DNS формат преобразуется в typed servers; сохраняются путь remote DNS
  через proxy, bootstrap/direct DNS, стратегии IP, FakeIP и predefined blocking.
  `outbound:any` bootstrap переносится в `default_domain_resolver`.
- WireGuard outbound преобразуется в endpoint, с сохранением tag, detour, адресов,
  ключей, reserved и peer routes. Встроенный WireGuard не удалён.
- TUN адреса объединяются; legacy sniff/resolve и UDP domain unmapping становятся
  предварительными route actions с сохранением порядка. Для прежней настройки
  sniff-and-override в патче добавлен соответствующий параметр sniff action.
- Перенесены Android platform API, асинхронный local DNS, connection-owner lookup,
  статистика TCP/UDP/нового TUN flow path и selector callback.
- GeoIP/Geosite pseudo paths и загрузка прежних баз сохранены. Старый unsafe gVisor
  shutdown workaround не переносится: в новой sing-tun уже есть корректный
  `Attach(nil)` и закрытие stack, подтверждённые чтением исходников.
- Приложение называется NekoBox SB и использует отдельный package для совместной
  установки с исходным клиентом. Ссылки проверки обновлений направлены на fork.
- Синяя тема выбрана по умолчанию, выбор другого цвета сохранён. Новая иконка:
  синий щит с белой N, включая adaptive/monochrome Android13+ и fallback Android21+.
  Launcher shortcuts получают package соответствующего build variant; reset
  notification направляется только своему package.
- Сброс активных соединений перенесён на `Network().ResetNetwork()`: метод Router
  очищает только кеши. Закрытие настоящего TCP-потока проверено отдельным тестом.

## REALITY и актуальный Xray

В исходном sing-box1.14.4 REALITY удаляет `X25519MLKEM768` из curves/key shares.
В REALITY, используемом Xray26.10.31, ClientHello без этой hybrid key share
отклоняется. Поэтому одной смены номера sing-box для такой совместимости мало.

В патче сохраняется hybrid key share выбранного fingerprint и используется
`MlkemEcdhe`, если отдельный ECDHE key отсутствует, согласно алгоритму клиента Xray.
Проверка подлинности REALITY сохраняется; проверки сертификатов не отключаются.
Заявленная проверенная комбинация: `fp=chrome`, RAW/TCP + REALITY + Vision.

Первоисточники:

- [sing-box1.14.4 REALITY client](https://github.com/SagerNet/sing-box/blob/v1.14.4/common/tls/reality_client.go)
- [Xray26.10.31 client](https://github.com/XTLS/Xray-core/blob/v26.10.31/transport/internet/reality/reality.go)
- [REALITY server check at pinned commit](https://github.com/XTLS/REALITY/blob/83f621788ca8/tls.go#L203)

На изолированном localhost стенде проверены подлинность outer REALITY и HTTPS
POST/echo через Vision с точным совпадением110592 байт. Серверы: Xray26.10.31
и26.2.6. Приватный ключ стенда генерируется в памяти, server JSON передаётся stdin.
Для localhost echo в тестовом Xray явно разрешён только127.0.0.1/32, поскольку
новый freedom по умолчанию блокирует private targets. Рабочие серверы не менялись.
Этот результат не является live-приёмкой телефона или доказательством причины
прежнего инцидента на пользовательском VPS.

## Native Windows сборка

Не требуется WSL/Git Bash. Нужны native Git, PowerShell7, Go1.25.5, JDK21 и SDK35
с build-tools35.0.1/NDK28.2.13676358. Пути ниже — пример; использовать свою установку.

```powershell
./buildScript/windows/Build-Android.ps1 `
  -SdkRoot 'C:\Android\Sdk' `
  -JavaHome 'C:\Java\jdk-21' `
  -Go 'C:\Go\bin\go.exe'
```

`Build-Libcore.ps1` закрепляет исходники, применяет патч и создаёт arm64 AAR.
`Prepare-Assets.ps1` восстанавливает отсутствующие Geo/Yacd assets из upstream1.4.2
APK с проверкой известного SHA256; существующие custom assets не перезаписывает.
`Build-Android.ps1` собирает `OssDebug`. После отдельно проверенного AAR возможен
`-SkipLibcore`. Кеши и временные файлы находятся в игнорируемом `.build`.

Сборка arm64 не доказывает работу других ABI. Gradle upstream создаёт также пустые
split APK для ABI, которых нет в AAR; использовать только arm64-v8a APK.
Публичный release signing key не создавался: alpha APK имеет debug подпись.
Для обновления установленного APK нужна та же подпись; сохранять локальный debug
keystore, перед stable использовать отдельно подготовленный постоянный ключ.
Windows scripts сохраняют переданный `ANDROID_USER_HOME`: для следующих alpha
сборок использовать тот же каталог с `debug.keystore`. Вложенная сборка libcore
не должна переключать этот каталог и тем самым менять подпись APK.

Существующие Linux GitHub workflows сохранены, Go закреплён, source script
применяет тот же patch. Их выполнение и остальные ABI требуют отдельной проверки.

## Проверки и границы

Финальный `OssDebug` arm64 APK собран02.11.2026. Подпись v1/v2 проверена
`apksigner verify`; `zipalign -c -P 16 4` PASS. В APK ровно одна native library,
arm64 libgojni.so, побайтно совпадающая с финальным AAR. Runtime label
`1.14.4-neko.1`, все четыре ELF PT_LOAD align16384. Bundled Geo/Yacd assets
присутствуют; package, версия, все локализованные labels `NekoBox SB`, variant
shortcut resource и launcher21/26/33 проверены в готовом APK.
Это статические проверки, не runtime-приёмка16KB/Android.

APK: `NekoBox-SB-1.4.3-sb1.14.4-alpha1-arm64-v8a.apk`.
SHA256: `9c68302466e65d97bf8c477fa3209698053939fd4197993070b7dd1a8c44ca78`.
Исходники и обсуждение: [draft PR1](https://github.com/silikonoff/NekoBoxForAndroid/pull/1).

Выполнены: baseline Go/libcore, baseline arm64 AAR/APK; compile нового libcore;
Windows-тесты счётчиков и route/rule; преобразование synthetic профилей и decode
их настоящим парсером1.14.4; localhost VLESS TCP/UDP; чтение bundled GeoIP/Geosite;
REALITY/Vision HTTPS с двумя указанными Xray; чистое применение compatibility patch;
PowerShell syntax и подготовка pinned sources.

Дополнительные команды:

```powershell
cd libcore
go test -tags 'with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api' ./configcompat
# Windows optional isolated Xray server test:
$env:NEKO_XRAY_BINARY = 'C:\Tools\Xray\xray.exe'
go test -count=1 -run TestXrayRealityVisionHTTPS -tags 'with_utls,with_quic,with_gvisor,with_wireguard,with_clash_api' ./configcompat
```

Не проверены: запуск/реальный VPN на TECNO; Wi-Fi↔mobile, sleep/Doze, reconnect,
DNS/IPv6 leaks, per-app routing и always-on/lockdown на Android; действующие
пользовательские подписки/серверы; WireGuard/SS/Hysteria/TUIC/plugin live traffic;
физическое16KB Android-устройство и прочие ABI. Новые Xray-specific VLESS encryption
и ML-DSA certificate extensions не являются заявленной функцией этого этапа.

## Приёмка на телефоне

1. Экспортировать резервную копию профилей из обычного NekoBox; установить SB рядом.
2. Импортировать копию, проверить группы, правила и обновление подписок.
3. Проверить реальный HTTPS upload/download через прежний VLESS/REALITY/Vision
   профиль и новый серверный Xray. Проверить внешний IP/DNS/IPv6 по выбранной политике.
4. Проверить direct/block/domain/per-app rules и FakeDNS; остальные используемые
   протоколы отдельно. Статус connected без трафика не считается PASS.
5. Проверить смену сети, screen-off и повторное подключение. До этих результатов
   alpha не заменяет рабочий клиент. Откат: открыть исходный NekoBox с его прежней БД.

Следующий этап после приёмки: отдельная интеграция Xray для XHTTP.

## Диагностическая alpha2 / 02.11.2026

Владелец сообщил timeout при «Проверить подключение» на TECNO KL7/Android14
по Wi-Fi; обычный NekoBox в этот момент выключен. Оба экспорта телефона
не содержат сетевой трассировки при default logLevel=0. Причина не установлена.

В изолированном Android14 x86_64 AVD alpha1 проходит локальные VLESS TCP,
DNS и REALITY/Vision HTTP/HTTPS echo. Без VPN контрольный TEST-NET адрес
даёт ожидаемый timeout. Приватный профиль владельца импортирован настоящим
парсером приложения: отдельный URL test PASS65ms, а независимый UID проходит
public HTTPS через Android VpnService, VPN=true/exit countryNL. Ключ нигде
не опубликован. Exact arm64 APK под libndk_translation Google APIs14 выдаёт
SIGILL при старте, поэтому ARM runtime этим эмулятором не принят.

Alpha2-diag добавляет автоматическую безопасную диагностику двух реальных
путей тестирования: TestInstance/profile (5000ms) и running VPN (3000ms).
Поведение HTTP-теста и тайм-ауты сохранены. `connectiontest.Trace` дополняет
HTTP trace без подмены timing callbacks; ошибки классифицируются фиксированными
токенами. Этап acquire_connection включает работу dialer/core, а не гарантированно
только TCP: отсутствие дальнейшего этапа само по себе не различает core DNS
и прокси-handshake. TLS-события относятся к HTTPS test target.

Диагностика ограничена32KiB на процесс main/bg и доступна при logLevel=0.
Обычный экспорт журнала включает её; очистка журнала удаляет её. В этих записях
нет адресов, URL, имён профилей, UUID, ключей и полного текста ошибок. Full core
и plugin config dumps удалены. Автоматической отправки данных нет.

ARM64 APK `NekoBox-SB-1.4.3-sb1.14.4-alpha2-diag-arm64-v8a.apk`, versionCode240,
прежние package и signing certificate, устанавливается поверх alpha1.
SHA256: `1cc04d85598848cac33ce4150a4ff78b041b8daac6bc4a1b8c746968f5d61c93`.
Подпись, ZIP/ELF16KiB, bundled assets и совпадение SO с новым AAR проверены.
После обновления x86_64 AVD профиль сохранился: URL test69ms, реальная кнопка
running VPN74ms, отдельный HTTPS probe200/VPN=true/NL821ms.

Это инструментализация неустранённого сбоя, не подтверждённое исправление
телефонного VPN. Настройки сервера, DNS и REALITY authentication не менялись.
Unit/regression: `go test -count=1 -tags 'with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api' ./configcompat ./connectiontest`.

## Alpha3: REALITY TCP packet compatibility

Новый телефонный экспорт alpha2 подтверждает два одинаковых отказа: TCP51ms,
затем timeout3000/3002ms без connection_ready/request_sent. При предоставленном
raw VLESS/REALITY профиле это локализует ожидание в установлении защищённого
outbound, а не в получении HTTP ответа. Конкретная причина на TECNO не доказана.

Воспроизведён отдельный дефект: RealityClientConfig.ClientHandshake не применял
fragment=true из базового UTLSClientConfig. `TestRealityHonorsTCPFragment` до
правки FAIL (first_write1785=whole_record1785), после PASS (частичный first write).
Патч применяет уже существующий tlsfragment.Conn к REALITY при fragment=true.
Разбиваются TCP-записи, а не TLS records; ClientHello/MLKEM/authentication bytes
сохраняются. Режим не включает record_fragment или spoof.

Runtime normalizer по умолчанию включает fragment=true/fallback50ms только для
VLESS с REALITY и без дополнительного транспорта. Явный fragment=false сохраняется,
остальные протоколы/транспорты не меняются. БД и подписки не переписываются.
Это проверяемый кандидат для телефонного пути, не установленная причина DPI.
Тайм-ауты app test3000/5000ms не увеличены. Ядро1.14.4-neko.2,
APK1.4.3-sb1.14.4-alpha3-reality/versionCode245, прежний package и signing key.

Нативные проверки: failing→passing fragment test, full configcompat/connectiontest,
REALITY/Vision HTTPS echo110592bytes с Xray26.2.6 и26.10.31 PASS; приватный NL
профиль с fragment=true проходит HTTPS204. Compatibility patch применён к clean
pinned index и воспроизвёл build source. Физический TECNO остаётся непринятым.

Android14/x86_64 alpha2→alpha3 install-r PASS, профиль сохранился; app profile
test78ms, настоящая running VPN button93ms. Отдельный UID: public HTTPS200,
VPN=true/NL2054ms. Синтетический REALITY/Vision: HTTP echo131072bytes1001ms,
UDP DNS539ms PASS (FakeDNS выключен, remote DNS идёт через proxy). Контролируемые
Android success/timeout/certificate и export/clear PASS; TLS rejection сохранён.

ARM64 APK25692269bytes, подпись прежняя, ZIP/ELF16KiB, SO=AAR, assets/icons PASS.
SHA256: `cdac847a32ceaa7e87ec0849e82d1c245d47845a44e9c2313d6c457958b296cd`.
Эти результаты не являются runtime-приёмкой на ARM64 TECNO и не подтверждают
гипотезу о фильтрации его ClientHello. Alpha3 остаётся кандидатом для проверки.

Продолжение: на физическом TECNO текущий профиль принят с `fp=edge`, включая
независимый DNS+HTTPS через VPN/NL. Chrome остаётся failing в полном VPN;
это не глобальная замена fingerprint. [Проверки и ограничение](TECNO_REALITY_COMPATIBILITY.md).
