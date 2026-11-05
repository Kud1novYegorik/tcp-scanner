# TCP Scanner

Библиотека на Go для многопоточного сканирования TCP-портов указанных хостов.

Позволяет проверять доступность портов, настраивать количество одновременных подключений и таймаут соединения, а также управлять разрешением DNS-адресов.

## Возможности

- Сканирование одного или нескольких хостов.
- Проверка отдельных портов, диапазонов и произвольных наборов портов.
- Ограничение количества параллельных проверок.
- Настройка таймаута TCP-подключения.
- DNS-разрешение хостов с возможностью проверки всех полученных IP-адресов.
- Поддержка `context.Context` для отмены сканирования.
- Потоковая выдача результатов через канал.
- Возможность подмены Dialer и Resolver для тестирования и кастомизации.

## Требования

- Go 1.25.5 или выше.

## Установка

После настройки корректного module path:

```bash
go get github.com/Kud1novYegorik/tcp-scanner
```

## Использование

### Простой пример

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Kud1novYegorik/tcp-scanner"
)

func main() {
    scanner, err := portscan.New()
    if err != nil {
        log.Fatal(err)
    }

    ports := portscan.Port(80, 443, 8080)

    results, err := scanner.Scan(
        context.Background(),
        []string{"example.com"},
        ports,
    )
    if err != nil {
        log.Fatal(err)
    }

    for result := range results {
        fmt.Printf(
            "%s:%d - %s\n",
            result.Host,
            result.Port,
            result.State,
        )
    }
}
```

### Сканирование диапазона портов

Для проверки диапазона используется функция `Range`:

```go
ports := portscan.Range(1, 1024)
```

### Объединение наборов портов

Можно объединять отдельные порты и диапазоны с помощью `Ports`:

```go
ports := portscan.Ports(
    portscan.Port(22, 80, 443),
    portscan.Range(8000, 8100),
)
```

### Создание набора портов из строки

Для разбора строки используется `ParsePorts`:

```go
ports, err := portscan.ParsePorts("22,80,443,8000-8100")
if err != nil {
    log.Fatal(err)
}
```

Поддерживаются отдельные порты, диапазоны и их комбинации через запятую.

### Настройка сканера

Параметры сканера задаются через функциональные опции:

```go
scanner, err := portscan.New(
    portscan.WithConcurrency(200),
    portscan.WithConnectTimeout(2*time.Second),
    portscan.WithResolveAll(true),
)
if err != nil {
    log.Fatal(err)
}
```

По умолчанию используются:
- 100 одновременных проверок;
- таймаут подключения 1 секунда;
- проверка всех IP-адресов, полученных при DNS-разрешении.

### Отмена сканирования

Для управления временем выполнения используется `context.Context`:

```go
ctx, cancel := context.WithTimeout(
    context.Background(),
    10*time.Second,
)
defer cancel()

results, err := scanner.Scan(
    ctx,
    []string{"example.com"},
    portscan.Range(1, 1000),
)
if err != nil {
    log.Fatal(err)
}

for result := range results {
    fmt.Println(result)
}
```

## Публичный API

### Scanner

Основная структура библиотеки, отвечающая за выполнение сканирования.

Создаётся функцией `New`.

```go
func New(opts ...Option) (*Scanner, error)
```

Принимает набор функциональных опций и возвращает настроенный экземпляр сканера. Если конфигурация некорректна, возвращает ошибку.

Основной метод:

```go
func (s *Scanner) Scan(
    ctx context.Context,
    hosts []string,
    ports PortSet,
) (<-chan Result, error)
```

Параметры:
- `ctx` — контекст для управления временем выполнения и отмены сканирования.
- `hosts` — список доменных имён или IP-адресов.
- `ports` — набор TCP-портов для проверки.

Возвращает:
- канал `<-chan Result`, из которого поступают результаты проверок;
- ошибку, если входные параметры некорректны или контекст уже отменён.

Результаты поступают асинхронно. Ошибки, возникающие при DNS-разрешении или проверке конкретного адреса, отражаются в соответствующем `Result`.

### Option

```go
type Option func(*config)
```

Функциональная опция для настройки сканера.

Доступные опции:

| Функция | Описание |
|---|---|
| `WithConcurrency(n int)` | Устанавливает максимальное количество одновременных проверок. |
| `WithConnectTimeout(d time.Duration)` | Устанавливает таймаут TCP-подключения. |
| `WithResolveAll(all bool)` | Определяет, проверять ли все IP-адреса, полученные при DNS-разрешении. |
| `WithResolver(r Resolver)` | Позволяет передать собственную реализацию DNS Resolver. |
| `WithDialer(d Dialer)` | Позволяет передать собственную реализацию Dialer. |

### PortSet

```go
type PortSet struct
```

Представляет набор TCP-портов для сканирования.

Создаётся с помощью следующих функций:

```go
func Port(ports ...int) PortSet
func Range(from, to int) PortSet
func Ports(parts ...PortSet) PortSet
func ParsePorts(s string) (PortSet, error)
```

Методы:

| Метод | Описание |
|---|---|
| `Err() error` | Возвращает ошибку, возникшую при создании набора портов. |
| `Len() int` | Возвращает количество портов в наборе. |
| `All() iter.Seq[uint16]` | Возвращает последовательность всех портов набора. |

Допустимый диапазон портов: от 1 до 65535.

### Result

```go
type Result struct {
    Host     string
    IP       net.IP
    Port     uint16
    State    State
    Duration time.Duration
    Err      error
}
```

Содержит результат проверки одного TCP-порта.

Поля:
- `Host` — исходное имя хоста.
- `IP` — IP-адрес, который проверялся.
- `Port` — проверяемый порт.
- `State` — состояние TCP-подключения.
- `Duration` — время выполнения проверки.
- `Err` — ошибка, связанная с проверкой или разрешением адреса.

### State

Тип `State` определяет результат проверки TCP-подключения.

```go
type State uint8
```

Возможные значения:

| Состояние | Описание |
|---|---|
| `StateOpen` | TCP-подключение успешно установлено. |
| `StateClosed` | Подключение отклонено удалённой стороной. |
| `StateTimeout` | Истекло время ожидания подключения. |
| `StateUnreachable` | Адрес или сеть недоступны. |
| `StateCanceled` | Проверка отменена через контекст. |
| `StateError` | Произошла другая ошибка. |
| `StateUnknown` | Неизвестное состояние. |

Для получения строкового представления состояния используется метод:

```go
func (s State) String() string
```

### Dialer

```go
type Dialer interface {
    DialContext(
        ctx context.Context,
        network string,
        address string,
    ) (net.Conn, error)
}
```

Интерфейс для выполнения TCP-подключений. Позволяет использовать собственную реализацию Dialer.

### Resolver

```go
type Resolver interface {
    LookupNetIP(
        ctx context.Context,
        network string,
        host string,
    ) ([]netip.Addr, error)
}
```

Интерфейс для DNS-разрешения хостов. Позволяет использовать собственную реализацию Resolver.

### Ошибки

Библиотека предоставляет следующие экспортируемые ошибки:

| Ошибка | Описание |
|---|---|
| `ErrInvalidPort` | Указан некорректный номер порта или диапазон. |
| `ErrNoPorts` | Передан пустой набор портов. |
| `ErrInvalidTarget` | Указан некорректный хост. |
| `ErrNoTargets` | Не переданы хосты для сканирования. |
| `ErrInvalidConfig` | Передана некорректная конфигурация сканера. |

Для проверки ошибок можно использовать `errors.Is`.

## Тестирование

Для запуска тестов:

```bash
go test ./...
```

