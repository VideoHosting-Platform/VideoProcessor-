# VideoProcessor-
Обработка видео (перекодирование в разные форматы)

# Как запускать:
- запустить rabbitmq (scripts/)
- запустить minIO (scripts/)
    - создать бакет : videos
    - загрузить какое-то видео(без .mp4) с именем в формате uuid (пример a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11 в общем
     videos/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11 )
- запустить cmd/main
- в rabbitmq в очереди отправить json сообщение(с uuid именем видео в minIO):
```json
{"video_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","user_id":123,"video_title":"My Awesome Video"}
```
- после, если не было ошибок, в minIO должна появится папка с обработанными(обновить сайт иногда надо)


## K8s
VideoProcessor - микросервис, не нуждается в service в k8s, т.к. его не вызвывают другие поды.

###  docker hub
valery223344/video_processor:0.0.1