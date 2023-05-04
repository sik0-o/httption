# httption
Actions that done on http transport

# BaseAction
BaseAction - является структурой реализующей HttpAction Interface. BaseAction выполняет заполненый в параметрах запрос и при успешном итоге возвращает результат запроса или же ошибку в случае неудачи. С помощью метода Repeat можно произвести запрос заново. В любом другом случае будет происходить setup реквеста перед его выполнением.


Основными параметрами BaseAction являются:
- http-client
- request method
- request url
- request headers

# Жизненый цикл BaseAction

```
Создание -> Выполнение Do -> Получение Result 
-> Обновление данных -> Выполнение Do -> Получение Result
```



