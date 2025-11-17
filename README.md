# Outbox Package

Пакет реализует паттерн Outbox для обеспечения надежной доставки событий через Kafka с использованием Postgres в качестве хранилища.

# Download

go get -u github.com/fedotovmax/outbox@{version}

## Поддерживаемые драйверы

- Kafka — [Sarama](https://github.com/IBM/sarama)
- Postgres — [pgx](https://github.com/jackc/pgx)

## Требуемая таблица в базе данных

```sql
create table if not exists events (
    id uuid primary key default gen_random_uuid(),
    aggregate_id varchar(100) not null,
    event_topic varchar(100) not null,
    event_type varchar(100) not null,
    payload jsonb not null,
    status varchar not null default 'new' check(status in ('new', 'done')),
    created_at timestamp default now(),
    reserved_to timestamp default null
);
```

## Пример создания

pgxtx.Manager и pgxtx.Extractor - [pgxtx](https://github.com/fedotovmax/pgxtx)

```go

// logger — ваш slog.Logger
// producer — ваш Kafka Producer

type Producer interface {
	GetInput() chan<- *sarama.ProducerMessage
	GetSuccesses() <-chan *sarama.ProducerMessage
	GetErrors() <-chan *sarama.ProducerError
}


// txManager — pgxtx.Manager
// extractor — pgxtx.Extractor

// config — outbox.Config

ob := outbox.New(logger, producer, txManager, extractor, config)
```

## Конструктор `New`

| Параметр | Тип               | Описание                               |
| -------- | ----------------- | -------------------------------------- |
| `l`      | `*slog.Logger`    | Логгер для вывода событий и ошибок     |
| `p`      | `Producer`        | Адаптер для публикации событий в Kafka |
| `txm`    | `pgxtx.Manager`   | Менеджер транзакций для Postgres       |
| `ex`     | `pgxtx.Extractor` | Для извлечения транзакций из контекста |
| `cfg`    | `Config`          | Конфигурация Outbox                    |

## Методы

| Метод         | Сигнатура                                                          | Описание                                                    |
| ------------- | ------------------------------------------------------------------ | ----------------------------------------------------------- |
| `AddNewEvent` | `AddNewEvent(ctx context.Context, ev CreateEvent) (string, error)` | Добавление нового события в таблицу Outbox                  |
| `Find`        | `Find(ctx context.Context, f FindEventsFilters) ([]*Event, error)` | Поиск событий с фильтрацией                                 |
| `Start`       | `Start()`                                                          | Запуск обработки событий                                    |
| `Stop`        | `Stop(ctx context.Context) error`                                  | Остановка обработки с ожиданием завершения текущих операций |
