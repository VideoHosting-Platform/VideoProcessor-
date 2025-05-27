docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
#5672 — порт для клиентов (Go клиент подключается сюда).
#15672 — веб-интерфейс.

