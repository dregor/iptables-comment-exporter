# iptables-comment-exporter

Prometheus-экспортер счётчиков пакетов/байт для правил iptables/ip6tables,
помеченных `--comment "<prefix><label>"`.

## Зачем

`iptables-nft` (современный backend iptables поверх ядра `nf_tables`, стоит
на большинстве актуальных дистрибутивов) не сохраняет `-m comment` как
нативный nftables-комментарий — любой `-m`-матч уходит в непрозрачный
xt-compat блок, невидимый для `nft list` и, соответственно, для любых
nftables-based экспортеров (`metal-stack/nftables-exporter` и т.п.). Старые
экспортеры на `python-iptables`/libiptc (`madron/iptables-exporter`) тоже не
работают — эта библиотека ходит через устаревший x_tables API, которого на
iptables-nft хостах просто нет.

`iptables-save -c` — единственное место, где комментарий и счётчики видны
одновременно, независимо от backend'а (nft или legacy). Этот экспортер
парсит именно его.

## Как это работает

На каждый скрейп `/metrics`:

1. Выполняется `iptables-save -c` (и `ip6tables-save -c`, если не отключено).
2. Из вывода берутся только строки правил с `--comment`, начинающимся на
   `--prefix` (по умолчанию `iptables-exporter`, без пробела в значении —
   кавычки с пробелом внутри в `ExecStart=` ломаются на старых systemd,
   например 219 на CentOS7). Метка — то, что идёт после prefix и
   необязательных пробелов/табов-разделителей.
3. Каждое такое правило превращается в две метрики-counter'а с лейблами
   `family` (ipv4/ipv6), `table`, `chain`, `label`.

Никакого собственного трекинга дельт — отдаются сырые абсолютные значения
из `iptables-save`, а `rate()`/`increase()` в Prometheus сами справляются
со скрейп-к-скрейп дельтами и сбросом счётчика (перезапуск iptables/reboot).

Пример правила в rules.v4, которое подхватит экспортер:

```
-A INPUT -j DROP -m comment --comment "iptables-exporter default_last_dropped"
```

даст на `/metrics`:

```
iptables_comment_rule_packets_total{family="ipv4",table="filter",chain="INPUT",label="default_last_dropped"} 89
iptables_comment_rule_bytes_total{family="ipv4",table="filter",chain="INPUT",label="default_last_dropped"} 1011
```

## Структура репозитория

```
main.go                        — точка входа: флаги, HTTP-сервер, graceful shutdown
internal/collect/collect.go    — запуск *tables-save с таймаутом и лимитом объёма вывода
internal/parse/rule.go         — тип Rule
internal/parse/parse.go        — разбор вывода *tables-save -c в []Rule
internal/render/render.go      — []Rule -> текст Prometheus exposition format
internal/handler/handler.go    — http.Handler для /metrics, склеивает collect+parse+render
systemd/*.service               — пример юнита для деплоя (лимиты CPU/RAM, минимум capabilities)
```

Каждый файл маленький и решает одну задачу — не нужно перечитывать/парсить
что-то большое, чтобы понять или поправить один шаг конвейера
(сбор -> парсинг -> рендер -> HTTP).

Конвейер данных: `handler.Metrics` на каждый запрос вызывает
`collect.Run("iptables-save", ...)` (и `ip6tables-save`, если не `--no-ipv6`)
-> результат отдаётся в `parse.Parse` -> получившиеся `[]parse.Rule` идут в
`render.Render` -> текст пишется в ответ.

## Флаги

| Флаг              | По умолчанию          | Описание                                              |
|--------------------|------------------------|--------------------------------------------------------|
| `--bind`           | `127.0.0.1:9631`       | адрес:порт для `/metrics`                              |
| `--prefix`         | `iptables-exporter`   | обязательный префикс `--comment` (без пробела на конце), без него правило игнорируется |
| `--timeout`        | `5s`                   | таймаут одного вызова `*tables-save` за скрейп          |
| `--no-ipv6`        | `false`                | не вызывать `ip6tables-save`                           |
| `--mem-limit-mb`   | `128`                  | мягкий лимит памяти рантайма Go (`debug.SetMemoryLimit`), `0` — не ограничивать |

## Безопасность и лимиты

- `collect.Run` ограничивает вывод `*tables-save` 32 МиБ (`collect.MaxOutput`)
  и обрывает вызов по таймауту через `context` — зависший или аномально
  большой вывод не свалит процесс.
- Скрейпы `/metrics` сериализуются мьютексом в `handler.Metrics` — параллельные
  запросы Prometheus не породят несколько одновременных `*tables-save`.
- `runtime.GOMAXPROCS(1)` и `debug.SetMemoryLimit` — программные лимиты CPU/RAM
  на случай аномалий в самом Go-рантайме.
- Пример systemd-юнита (`systemd/iptables-comment-exporter.service`) добавляет
  лимиты на уровне ОС: `MemoryMax`, `CPUQuota`, `TasksMax`, `NoNewPrivileges`,
  `ProtectSystem=strict`.
- Запуск без root (`User=prometheus` + `CAP_NET_ADMIN`/`CAP_NET_RAW` через
  `AmbientCapabilities`) технически возможен и работает на части хостов, но
  ненадёжно: `iptables-save`/`ip6tables-save` (nftables backend) иногда
  отказывают в "fetch rule set generation id: Permission denied (you must be
  root)" даже с этими capability через Ambient — воспроизведено на разных
  комбинациях ОС/ядра/nftables без чёткого правила, когда именно. Надёжно
  работает только полный root без ограничения capability — так и в примере
  юнита.
- Все внешние команды запускаются через `exec.CommandContext` с фиксированными
  аргументами (без shell), инъекция через ввод невозможна.

## Сборка и тесты

```
make build   # статический бинарник ./iptables-comment-exporter
make test    # go test ./...
make vet
```

## Локальный запуск

```
./iptables-comment-exporter --bind=127.0.0.1:9631
curl http://127.0.0.1:9631/metrics
```

Для чтения `iptables-save`/`ip6tables-save` на большинстве систем нужен
root или `CAP_NET_ADMIN`/`CAP_NET_RAW` — см. systemd-юнит.
