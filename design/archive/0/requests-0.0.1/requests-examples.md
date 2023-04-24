### Просмотр проекции `inbox` (несортированные)

`<user>/api/<wsid>/view/inbox`

```
{
    "from": 123,
    "to": 456,
    "page" 3
}
```

### Просмотр проекции `archive` (архив)
`<user>/api/<wsid>/archive`


### Просмотр проекции `spam` (спам)
`<user>/api/<wsid>/spam`


### Просмотр проекции `sent` (отправленные)
`<user>/api/<wsid>/spam`


### Просмотр проекции `sent` (отправленные)
`<user>/api/<wsid>/spam`

### Просмотр проекции `contacts`(адресной книги)
`<user>/api/<wsid>/contacts`

### Создание объекта класса `email`
`<user>/api/<wsid>/email` 

```
{
    "from": 'yohanson@hmail.com',
    "to": 'gmp@hmail.com,smm@gmail.com',
    "bcc": 'ivs@gmail.com',
    "theme": 'Updated theme',
    "content": "..."
    "tags": [ ... ],
    "draft": null, 
    "send_date": 1575832455,
}
```

### Изменение объекта класса `email`
`<user>/api/<wsid>/email` 

```
{
    "id": 123,
    "theme": 'Updated theme',
    "content": "New content"
}
```

### Получение данных объекта класса `email`
`<user>/api/<wsid>/email/<email id>` 

### Метод - изменяет статус email на 2 (удален)

`<user>/api/<wsid>/email/status` 

```
{
    "id": 987654321,
    "status": 2
}
```

### Метод - отправка почтового сообщения
`<user>/api/<wsid>/email/send` 

```
{
    "id": 123
}
```

### Соаздание/изменение объекта класса `accaunt`
`<user>/api/<wsid>/accaunt`

### Метод - удаление аккаунта
`<user>/api/<wsid>/accaunt/remove`

### Создание\изменение конфигурации

Так как `config` определен как класс-синглетон, то для изменения его параметров не требует передавать id сущности

`<user>/api/<wsid>/config`

```
{
    "color": "#fff",
    "format": "dd.mm.yyyy"
}
```

## Функции

### Добавить тег

`<user>/api/<wsid>/addTag`

```
{
    "tag": '<tag code>',
    "id": 13245, 
}
```

### Удалить тег
`<user>/api/<wsid>/removeTag`

```
{
    "tag": '<tag code>',
    "id": 13245, 
}
```

### Поиск по почте
`<user>/api/<wsid>/search`

```
{
    "text": 'Search string',
}
```

### Массовое изменение состояния
`<user>/api/<wsid>/changeState`

```
{
    "id": [ <id>, <id>, ... ],
    "state": 2
}
```