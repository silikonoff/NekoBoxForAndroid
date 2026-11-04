# Downloads — silikbox для Android

Готовые подписанные APK. Собирать приложение самостоятельно не требуется.

**Для последних стабильных [3x-ui 3.11.0](https://github.com/MHSanaei/3x-ui/releases/tag/v3.11.0) и [Xray 26.10.31](https://github.com/XTLS/Xray-core/releases/tag/v26.10.31)** по состоянию на 04.11.2026: **VLESS TCP/RAW + REALITY + XTLS Vision**, включая исходные Chrome-профили. Совместимость механизма проверена в нашей alpha5 и сохраняется в silikbox.

## Актуальная версия: 1.0.0-alpha2

### [⬇ Скачать silikbox для Android · ARM64](https://github.com/silikonoff/silikbox/releases/download/v1.0.0-alpha2/silikbox-1.0.0-alpha2-arm64-v8a.apk)

| Файл | Назначение |
| --- | --- |
| **[silikbox-1.0.0-alpha2-arm64-v8a.apk](https://github.com/silikonoff/silikbox/releases/download/v1.0.0-alpha2/silikbox-1.0.0-alpha2-arm64-v8a.apk)** | Установка или обновление Android-приложения |
| [silikbox-1.0.0-alpha2-SHA256.txt](https://github.com/silikonoff/silikbox/releases/download/v1.0.0-alpha2/silikbox-1.0.0-alpha2-SHA256.txt) | Контрольная сумма APK |
| [silikbox-1.0.0-alpha2-validation.json](https://github.com/silikonoff/silikbox/releases/download/v1.0.0-alpha2/silikbox-1.0.0-alpha2-validation.json) | Метаданные и результаты проверки артефакта |

**[Описание выпуска →](https://github.com/silikonoff/silikbox/releases/tag/v1.0.0-alpha2)** · **[Все версии →](https://github.com/silikonoff/silikbox/releases)** · **[Возможности приложения →](README.md)**

## Что выбрать

- Готовая сборка предназначена для **ARM64 / arm64-v8a**. Для других архитектур APK сейчас не публикуются.
- Минимальный API в проекте — **Android 5.0 / API21**; проверка механизма REALITY на физическом устройстве выполнена на **Android14**. Приёмка на всех версиях Android ещё не завершена.
- Версия **alpha** предназначена для раннего использования и обратной связи. Известные ограничения и проверки указаны в описании конкретного релиза.
- Установочный файл имеет расширение **`.apk`**. GitHub «Source code» — архив исходников. Неподписанный артефакт Actions предназначен для проверки сборки.

## Установка и обновление

1. Скачайте APK по кнопке выше и откройте его на телефоне.
2. Разрешите установку из выбранного источника, если Android запросит это разрешение.
3. При обновлении нашей прежней версии установите APK **поверх неё**. Пакет и сертификат NekoBox SB / NickMadBox сохранены, versionCode новой сборки — **265**.
4. Импортируйте сервер или подписку, выберите профиль и подтвердите системное разрешение на VPN при первом подключении.

Оригинальный NekoBox может оставаться установленным рядом с silikbox. Одновременно Android использует одно VPN-подключение.

Текущая версия использует **sing-box 1.14.4** с нашим патчем совместимости. **XHTTP / SplitHTTP ещё не поддерживается**; его добавление с Xray запланировано следующим этапом.
