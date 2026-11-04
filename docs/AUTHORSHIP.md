# Авторство и независимое сопровождение

NickMadBox — самостоятельная версия и дальнейшие доработки **silikonoff**.
Репозиторий, релизы, обновления и обращения пользователей:
https://github.com/silikonoff/NickMadBox.

Основа — NekoBox for Android/MatsuriDayo, commit
`5768494d8ae3c74a057bb6d46c0f8dc071b0d821`, и предшествующий SagerNet/nekohasekai.
История Git, уведомления авторов и GPL-3.0-or-later сохранены. Наше авторство
относится к своим изменениям и сопровождению, а не к чужим исходникам.

## Наши доработки

Перенос на sing-box1.14.4, совместимость конфигурации и Android API,
автоматическая совместимость REALITY, синяя тема и отдельная иконка,
обособленный пакет, русская версия NickMadBox и собственный процесс сборки.

## Закреплённые зависимости

| Компонент | Источник и версия |
| --- | --- |
| sing-box | SagerNet/sing-box, `c005f35300ce76427dc285481ec5695f5b179cd2` |
| Наш патч ядра | `buildScript/patches/nekobox-core.patch` |
| libneko | MatsuriDayo/libneko, `1c47a3af71990a7b2192e03292b4d246c308ef0b`; снимок в `third_party/libneko` |
| gomobile | MatsuriDayo/gomobile, `17d6af34f6bd6d7e1e428e0c652c8b54a46bda4f`; снимок в `third_party/gomobile` |

Снимки сохраняют исходные уведомления. gomobile основан на Go Mobile;
LICENSE/AUTHORS/CONTRIBUTORS включены рядом с кодом. Внешние Go/Maven-библиотеки
и Geo-базы остаются зависимостями со своими лицензиями. Независимое сопровождение
не означает отказ от экосистемы открытого ПО. GitHub может показывать происхождение
fork: это связь истории, а не зависимость выпуска от исходного NekoBox.
