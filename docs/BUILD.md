# Сборка NickMadBox

Проверенный набор: Windows PowerShell, Python3.11+, Git, Go1.25.5, JDK21,
Android SDK35, build-tools35.0.1, NDK28.2.13676358. Путь проекта без пробелов.
В PATH должны быть Python, Git, Go и Java. Ядро — закреплённый sing-box с нашим
патчем; libneko/gomobile уже находятся в `third_party`.

```powershell
python tools/prepare_assets.py
./buildScript/windows/Build-Libcore.ps1 -SdkRoot C:/Android/sdk -JavaHome C:/Java/jdk-21
./gradlew.bat --no-daemon --max-workers=2 app:assembleOssDebug
```

В локальном `local.properties` задаётся `sdk.dir`; файл не отправляется в Git.
По умолчанию ядро ARM64: распространять только APK `arm64-v8a`.
`-UseExistingEnvironment` разрешает использовать настроенные GOPATH/GOCACHE/
GOTMPDIR; иначе сборщик создаёт их в `.build/windows`.

## Подпись и обновления

Пакет alpha `io.github.silikonoff.nekobox.debug` сохранён для обновления поверх
наших сборок. У оригинального NekoBox другой пакет. Kotlin namespace — внутреннее
имя кода, для отдельной установки его менять не требуется.

Подписанные релизы собираются локально прежним ключом владельца. Для обновления
поверх alpha5 нужны тот же пакет, сертификат и versionCode выше255.
NickMadBox1.0.0-alpha1 имеет versionCode260. Рабочий keystore не заменять и не
передавать в GitHub. SDK, кэши и AAR в исходники не входят.

GitHub Actions проверяет тесты и создаёт **неподписанный APK**. Это не установочный
релиз. Передачи ключа подписи, удаления релизов и отправки в Google Play нет.

## Проверки

```powershell
python tools/check_repository.py
go -C libcore test -tags with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api ./configcompat ./connectiontest
python tools/verify_apk.py APK_PATH app/libs/libcore.aar report.json --runtime-label 1.14.4-nickmadbox.1
```

Также проверять `apksigner verify`, alignment16KiB, пакет и ярлык через
`aapt dump badging`. Сборка не заменяет запуск VPN: нужны импорт, реальные
DNS/HTTPS, перезапуск и сохранение профилей. Публичные отчёты не должны содержать
ключи, пользовательские UUID, частные адреса серверов, serial и сырые журналы.
