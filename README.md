# go-batch

Создание сервиса прокси на Go, который будет батчить запросы к внешней системе, обеспечивая повышенную надежность и обработку ошибок, можно реализовать с использованием паттерна "Batch Processor".

Есть внешняя система, которая находится далеко и имеет не самый стабильный выход в интернет.
Нужно по запросу от клиентов ходить в эту систему, не забывая про таймауты и ретраи, и выдавать ответ клиенту, гарантируя хороший SLA (минимум ошибок в ответах).
Клиенты готовы к повышенным таймаутам ответа.

Пишем сервис прокси, которые позволяет скрыть большой процент ошибок при запросах во внешнюю систему.

При этом API внешней системы устроен так, что мы можем в одном API вызове передать запросы от нескольких клиентов, и на все запросы получить ответы за один сетевой поход. Если такой групповой запрос вернул ошибку, то повторяем всю группу целиком.
