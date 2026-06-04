# L4.5 Оптимизация простого API-сервиса с профилировкой

## Цель проекта

Цель проекта — взять простой HTTP API-сервис, создать для него нагрузку, снять профили CPU и памяти, выполнить анализ через `pprof`, `go test -bench`, `benchstat` и `go tool trace`, а затем оптимизировать горячий путь обработки запроса.

В качестве основы был взят сервис из L0 — API получения заказа по идентификатору. В L0 использовались Kafka, PostgreSQL, локальный кэш и HTTP API. Для задания L4.5 была сделана упрощённая версия сервиса, сфокусированная на HTTP-запросе:

```text
GET /order?id=order-1
```

Сервис хранит сгенерированные заказы в памяти, использует локальный кэш и возвращает заказ в формате JSON.

## Что реализовано

В проекте реализованы:

* HTTP API для получения заказа по ID;
* генерация тестовых заказов через `gofakeit`;
* in-memory storage, имитирующий базу данных;
* локальный кэш заказов;
* подключение `net/http/pprof`;
* генератор нагрузки;
* benchmark через `go test -bench`;
* сравнение результатов через `benchstat`;
* снятие CPU, heap и trace-профилей;
* оптимизация горячего пути обработки запроса.

## Структура проекта

```text
L4.5/
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── load/
│       └── main.go
├── internal/
│   ├── generator/
│   │   └── generator.go
│   ├── httpapi/
│   │   ├── handler.go
│   │   └── handler_test.go
│   ├── model/
│   │   └── order.go
│   └── repository/
│       └── repository.go
├── profiles/
│   ├── before_bench.txt
│   ├── json_cache_bench.txt
│   ├── after_bench.txt
│   ├── benchstat.txt
│   ├── before_cpu.pb.gz
│   ├── before_cpu_top.txt
│   ├── before_heap.pb.gz
│   ├── before_heap_top.txt
│   ├── before_trace.out
│   ├── before_load.txt
│   ├── after_cpu.pb.gz
│   ├── after_cpu_top.txt
│   ├── after_heap.pb.gz
│   ├── after_heap_top.txt
│   ├── after_trace.out
│   └── after_load.txt
├── go.mod
├── go.sum
└── README.md
```

## Запуск API

```bash
go run ./cmd/api
```

После запуска доступны:

```text
http://localhost:8080/order?id=order-1
http://localhost:8080/health
```

Пример запроса:

```bash
curl "http://localhost:8080/order?id=order-1"
```

Пример запроса к несуществующему заказу:

```bash
curl "http://localhost:8080/order?id=unknown"
```

## pprof

Для профилирования подключён стандартный пакет:

```go
_ "net/http/pprof"
```

pprof-сервер запускается отдельно от основного API:

```text
http://localhost:6060/debug/pprof/
```

Основной API работает на порту `8080`, а pprof — на порту `6060`.

## Генератор нагрузки

Для создания нагрузки реализована отдельная утилита:

```bash
go run ./cmd/load -url "http://localhost:8080/order?id=order-1" -c 50 -n 300000
```

Параметры:

```text
-url      адрес API
-c        количество параллельных клиентов
-n        общее количество запросов
-timeout  timeout одного запроса
```

Пример вывода:

```text
Load test finished
URL:          http://localhost:8080/order?id=order-1
Requests:     300000
Concurrency:  50
OK:           300000
Errors:       0
Duration:     ...
RPS:          ...
Avg latency:  ...
```

## Benchmark

Для измерения производительности обработчика используется benchmark:

```bash
go test ./internal/httpapi -run=^$ -bench=BenchmarkGetOrderCacheHit -benchmem -count=10 > profiles/before_bench.txt
```

Benchmark проверяет основной сценарий:

```text
GET /order?id=order-1
```

В benchmark измеряется cache hit-сценарий, то есть заказ уже находится в кэше. Это основной быстрый путь работы API.

## Базовая версия

В базовой версии обработчик работал так:

```text
HTTP request
→ получить id из query
→ найти заказ в repository cache
→ выполнить json.Marshal(order)
→ записать JSON в ResponseWriter
```

Проблемы базовой версии:

* `json.Marshal` выполнялся на каждый успешный запрос;
* каждый ответ создавал новый JSON-массив байтов;
* в горячем пути было подробное логирование;
* при высокой нагрузке часть CPU тратилась на сериализацию и логирование.

## Снятие профилей before

CPU-профиль:

```bash
curl -o profiles/before_cpu.pb.gz "http://localhost:6060/debug/pprof/profile?seconds=30"
```

Heap-профиль:

```bash
curl -o profiles/before_heap.pb.gz "http://localhost:6060/debug/pprof/heap"
```

Trace:

```bash
curl -o profiles/before_trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
```

Текстовый вывод CPU top:

```bash
go tool pprof -top profiles/before_cpu.pb.gz > profiles/before_cpu_top.txt
```

Текстовый вывод heap top:

```bash
go tool pprof -top profiles/before_heap.pb.gz > profiles/before_heap_top.txt
```

Просмотр trace:

```bash
go tool trace profiles/before_trace.out
```

## Анализ before

В CPU-профиле до оптимизации были заметны:

```text
encoding/json.Marshal
encoding/json.structEncoder.encode
encoding/json.appendString
log.Printf
log.(*Logger).output
net/http
syscall
runtime
```

Это показывало, что при повторных запросах сервис каждый раз заново сериализует структуру `Order` в JSON и выполняет логирование в горячем пути.

Heap-профиль показывал, что большая часть живой памяти занята заранее сгенерированными заказами. Поэтому для оценки временных аллокаций на один запрос использовался не только heap-профиль, но и `go test -benchmem`.

## Оптимизация 1. Кэширование готового JSON

Первая оптимизация — добавление кэша готовых JSON-ответов.

Было:

```text
cache hit Order
→ json.Marshal(order)
→ w.Write(data)
```

Стало:

```text
json cache hit
→ w.Write(data)
```

То есть после первого запроса к конкретному заказу результат сериализации сохраняется в `JSONCache`:

```go
type JSONCache struct {
    mu    sync.RWMutex
    items map[string][]byte
}
```

После этого повторные запросы к тому же заказу больше не выполняют `json.Marshal`.

Эта оптимизация относится к уровню HTTP, а не repository, потому что repository отвечает за бизнес-данные `model.Order`, а JSON — это формат HTTP-ответа.

## Промежуточный benchmark после JSON-кэша

После добавления JSON-кэша был снят промежуточный benchmark:

```bash
go test ./internal/httpapi -run=^$ -bench=BenchmarkGetOrderCacheHit -benchmem -count=10 > profiles/json_cache_bench.txt
```

Результат показал, что основное улучшение дала именно эта оптимизация.

## Оптимизация 2. Уменьшение логирования в горячем пути

Вторая оптимизация — отключение подробного логирования при обычной работе API.

Было:

```go
log.Printf("request order id=%s", id)
```

на каждый запрос.

Стало:

```go
func (h *Handler) logf(format string, args ...any) {
    if h.debug {
        log.Printf(format, args...)
    }
}
```

По умолчанию `debug = false`, поэтому при обычной нагрузке обработчик не пишет лог на каждый запрос.

Также логирование cache hit/cache miss было убрано из repository, чтобы repository не создавал лишнюю нагрузку в горячем пути.

## Финальный benchmark

После оптимизаций был снят финальный benchmark:

```bash
go test ./internal/httpapi -run=^$ -bench=BenchmarkGetOrderCacheHit -benchmem -count=10 > profiles/after_bench.txt
```

## Сравнение через benchstat

Сравнение выполнялось командой:

```bash
benchstat profiles/before_bench.txt profiles/after_bench.txt > profiles/benchstat.txt
```

Результат:

```text
sec/op:
6213.5 ns/op → 473.9 ns/op
улучшение: -92.37%

B/op:
3041 B/op → 448 B/op
улучшение: -85.27%

allocs/op:
9 allocs/op → 5 allocs/op
улучшение: -44.44%
```

## Снятие профилей after

CPU-профиль после оптимизации:

```bash
curl -o profiles/after_cpu.pb.gz "http://localhost:6060/debug/pprof/profile?seconds=30"
```

Heap-профиль после оптимизации:

```bash
curl -o profiles/after_heap.pb.gz "http://localhost:6060/debug/pprof/heap"
```

Trace после оптимизации:

```bash
curl -o profiles/after_trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
```

Текстовый вывод:

```bash
go tool pprof -top profiles/after_cpu.pb.gz > profiles/after_cpu_top.txt
go tool pprof -top profiles/after_heap.pb.gz > profiles/after_heap_top.txt
```

## Анализ after

После оптимизации `encoding/json.Marshal` перестал быть заметным узким местом в CPU-профиле при повторных cache hit-запросах.

Основная нагрузка сместилась в:

```text
net/http
syscall
bufio
runtime
чтение и запись сетевых соединений
```

Это означает, что после оптимизации горячий путь бизнес-логики стал значительно дешевле, а основная стоимость обработки запроса теперь связана с самим HTTP-сервером и сетевым вводом-выводом.

Heap-профиль после оптимизации также в основном показывает память, занятую сгенерированными заказами. Поэтому основное снижение временных аллокаций видно именно в `benchstat`:

```text
3041 B/op → 448 B/op
9 allocs/op → 5 allocs/op
```

## Анализ trace

Trace снимался до и после оптимизации:

```bash
go tool trace profiles/before_trace.out
go tool trace profiles/after_trace.out
```

Trace использовался для проверки поведения приложения под нагрузкой:

* сервис обрабатывает много параллельных HTTP-запросов;
* запросы обслуживаются goroutine HTTP-сервера;
* после оптимизации горячий путь обработчика стал короче;
* основная активность связана с сетевым вводом-выводом и работой `net/http`;
* долгих блокировок в бизнес-логике обработчика не наблюдалось.

## История оптимизации

Пример истории коммитов проекта:

```text
L4.5: add baseline order API
L4.5: add pprof endpoints
L4.5: add benchmarks for order handler
L4.5: add load generator
L4.5: save baseline profiling results
L4.5: cache encoded JSON responses
L4.5: reduce logging on hot path
L4.5: compare benchmarks with benchstat
L4.5: save optimized profiling results
L4.5: document profiling results in README
```

## Вывод

В результате работы был создан простой HTTP API-сервис получения заказа по ID, подключены инструменты профилирования и проведена оптимизация горячего пути обработки запроса.

Главная проблема базовой версии заключалась в том, что при каждом повторном запросе сервис заново сериализовал структуру `Order` в JSON. После добавления кэша готовых JSON-ответов повторные cache hit-запросы стали обрабатываться без повторного вызова `json.Marshal`.

Итоговые результаты:

```text
Время обработки:
6213.5 ns/op → 473.9 ns/op
улучшение на 92.37%

Память на операцию:
3041 B/op → 448 B/op
улучшение на 85.27%

Количество аллокаций:
9 allocs/op → 5 allocs/op
улучшение на 44.44%
```

После оптимизации основная нагрузка сместилась из бизнес-логики обработчика в стандартный HTTP-сервер и системные вызовы чтения/записи. Это показывает, что оптимизация успешно устранила основное узкое место в обработке cache hit-запросов.
