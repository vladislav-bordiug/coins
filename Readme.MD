# Как запустить

Перейти в корень проекта и выполнить:

```docker-compose up```

# Стек

Для выполнения использовался следующий стек:
+ Go
+ PostgreSQL
+ Docker Compose

У каждого пользователя по умолчанию 300 монет.

# Тесты и покрытие

Написаны e2e тесты для всех 4 сценариев.
Покрытие 80%+ у unit-тестов. e2e тесты в [e2e](e2e). 

![img.png](img/img.png)

# Линтер

Линтер описан в [.golangci.yml](./.golangci.yml)

Все успешно выполняется, ошибок нету (все были исправлены):

![img.png](img/img_1.png)

# Нагрузочное тестирование

Было выполнено нагрузочное тестирование, используя Vegeta и такие команды:

```vegeta attack -duration=5s -rate=10 -targets=D:\avitotest\vegetatests\auth\authtest.list -output=results.bin```

```vegeta plot -title=AttackResults results > results.html```

Файлы для тестирования и результаты в [vegetatests](vegetatests).

Контейнеры были развернуты с Docker Desktop.

/api/auth:

![vegeta-plot.png](img/vegeta-plot.png)

/api/info:

![vegeta-plot.png](img/vegeta-plot1.png)

/api/sendCoin (ошибки связаны с недостаточным кол-вом монет для отправки):

![vegeta-plot.png](img/vegeta-plot2.png)

/api/buy/{item} (ошибки связаны с недостаточным кол-вом монет для покупки):

![vegeta-plot.png](img/vegeta-plot3.png)