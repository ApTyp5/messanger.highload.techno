Курсовой проект "NotWatsApp"
==========




Аудитория: 
---------
50 млн человек по России [(половина пользователей телеграмма 4 года назад)](https://telegram.org/blog/100-million)




Планируемые нагрузки:
----------------
[100 миллионов пользователей](https://telegram.org/blog/100-million) были способны создавать нагрузку в 
[15 миллиардов запросов](https://telegram.org/blog/15-billion) в сутки

Следовательно для 50 млн пользователей нагрузка будет равна 
(15 * 10^12) / 2 = __7.5 * 10^12 (request per day)__

Но нагрузка распределена не равномерно, в соответствии со [статистикой](https://popsters.ru/blog/post/aktivnost-auditorii-v-socialnyh-setyah-issledovanie-2019) 
в пиковый час телеграм обрабаывает 4,5 % дневного трафика.

Следовательно, в пик нагрузки сервису приходится обрабаывать 
(7.5 * 10^12 * 4,5%) / 100% = __3 375 * 10^8 (rph)__

Предположив,что в течение часа/минуты нагрузка распределяется равномерно, расчитаем более точную нагрузку на сервис: 
```
  3 375 * 10^8 (request per hour) =
  = 56,25 * 10^8 (rpm) =
  = 93,75 * 10^6 (rps)
```


Учитывая, что в худшем случае каждому сообщению будет соответствовать запрос полученных сообщений пользователя то: 

1. Отправка сообщений займет _93,75 * 10^6 ~= 100 * 10^6_
2. Запрос полученных сообщений = _93,75 * 10^6 ~= 100 * 10^6_

В течение пикового часа возможны возникновения локальных пиков нагрузок, под которые следует выделить 

1. 2 * 100 * 10^6 = 200 * 10^6 на запись
1. 2 * 100 * 10^6 = 200 * 10^6 на чтение






Идея функционирования сервиса:
--------------------
1. К к каждому пользователю формируется очередь сообщений;
2. Пользователь может в любой момент вычитать пришедшие к нему сообщения;
3. История сообщений хранится на клиентах.





Схема бд:
--------------------
![Схема бд](https://github.com/ApTyp5/messanger.highload.techno/blob/main/schem.jpg)







Выбор конкретной СУБД
-----------------------------
В качестве СУБД выбрана Redis благодаря следующим характеристикам:
1. Репликация из коробки;
2. Мастабируемость из коробки;
3. Встроенная работа со списками.


Физическая особенность хранения очередей сообщений:
1. На каждого пользователя выделяется список, ключом которого является id пользователя.
2. В списке хранятся строковые записи.
3. Каждое отдельное сообщение состоит из 3-х элементов списка (id автора, timestamp, text).
4. После считывания очередь удаляется.





Кластеризация: шардинг, репликация 
----------------------------
 * Шардинг будет производиться по полю user_id;
 * Каждого мастера будут страховать 2 слейва;
 * Конфигурация кластера хранится на каждой из нод.
 
 
 
 
 
 
 
 
Балансировка
--------------------------
L4 балансировка будет производиться на уровне DNS - сервер будет отдавать virtual ip L7 балансировщиков
по алгоритму round-robin. Таким образом будет балансироваться нагрузка на балансировщики.

L7 балансировки будет производиться при помощи обратных прокси - сервер будет проксировать запрос на 
один из application-серверов по алгоритму round-robin. Таким образом все app-сервера будут одинаково 
нагружены.
 
 




Замеры времени вставки одного сообщения
------------------------------
![Замеры вставки одного сообщения](https://github.com/ApTyp5/messanger.highload.techno/blob/main/insert-bench.png)

Длина сообщения была равна 35 русских символов c пробелами, что соответствует 
[средней длине сообщения человека в возрасте 25 лет](https://crushhapp.com/blog/k-wrap-it-up-mom).

Округлим до 30 000 rps









Замеры времени получения одной очереди сообщений
------------------------------------ 
![Замеры вставки одного сообщения](https://github.com/ApTyp5/messanger.highload.techno/blob/main/read-bench.png)

В очереди сообщений при замерах находится 20 писем.

Округлим до 30 000 rps.







Выбор прочих технологий:
-------------------------------
1. ЯП: golang. Преимущества: быстрая разработка, встроенная многопоточность, большое сообщество.
2. Фреймворк: echo. Преимущества: популярный, простой фреймворк с хорошей документацией.
2. Протоколы взаимодействия: https при установлении соединения, после wss. Преимущества второго: позволяет 
реализовать real-time отправку сообщений наиболее естественным образом, без хаков вроде long polling.
3. Веб-сервер: nginx. Так как он обладает высокой проиводительностью и легко настраиваем.







Информация об эффективности Appliction серверов:
------------------------------------
Судя по [исследованиям](https://github.com/smallnest/go-web-framework-benchmark), прокидывание запросов к СУБД из golang и последующий возврат ответа
даёт в среднем результаты 30 000 rps.








Балансировщики нагрузки:
--------------------------------------
Исходя из [официальных бенчмарков](https://www.nginx.com/blog/nginx-websockets-performance/), 
nginx на 1 ядре спокойно держит до 50 000 подключений.

Особенности балансировки: DNS отдают адреса балансировщиков по round-robin с ttl 5min. таким образом при выходе из строя одного балансировщика
достаточно будет удалить его из записей DNS. Ситуация выхода из строя балансировщиков будет мониториться самописными heartbeat-демонами на DNS серверах.





Слово о DNS серверах:
---------------------
Так как DNS многократно кэшируется, то высоконагруженной её нельзя назвать. Но в нашем
случае DNS сервера будут обеспечивать мониторинг доступности балансировщиков (подробнее 
в обеспечении доступности). Однако для этого
также не нужно много ресурсов, поэтому для этих серверов подойдут 4-хядерные
8-гигабитные машины.


Требуемое количество ядер для различных категорий машин
----------------------------------------
|Категория сервера|Вычисления|Количество ядер|
|---------|----------|:---------------:|
|Мастер-ноды|(200 * 10^6 rps) / (30 * 10^3 rps)|6 667|
|Слейв-ноды|<количество ядер на master-node> * 2|неизвестно|
|App-серверы|(400 * 10^6 rps) / (30 * 10^3 rps)|13 334|
|Balancers|(400 * 10^6 rps) / (50 * 10^3 rps)|8 000|
|DNS| - | 24|




Взаимодействие всех частей в целом:
--------------------------------
Концептуально сервис будет работать следующим образом
![Схема бд](https://github.com/ApTyp5/messanger.highload.techno/blob/main/alll.jpg)


Рассчет потребного оборудования:
----------------------------------
Для серверов будут использованы стандартные 64-х ядерные сервера (24 для балансировщиков), содержащие
либо 32, либо 64 гигабайта памяти. Также будет учтен небольшой запас на случай 
непредвиденных обстоятельств.

|Категория сервера|Вычисления|Количество серверов| Учитываем запас|
|---------|----------|:---------------:|:-:|
|Мастер-ноды|6 667 / 64| 105| 110|
|Слейв-ноды|105 * 2|210| 210|
|App-серверы|13 334 / 64| 209| 220|
|Balancers|192 000 / 24| 8000| 8020|
|DNS|24 / 8|3|3|



Итог оснащения Сервиса:
-------------------------------
|Категория сервера|CPU(cores)|RAM(GB)| Количество|
|---------|----------|:---------------:|:-:|
|Мастер-ноды|64|64| 110|
|Слейв-ноды|64|64| 210|
|App-серверы|64|32| 220|
|Balancers|24|32| 8020|
|DNS|4|8|3|



Хостинг / облачный провайдер:
--------------------------------
Так как большая часть населения России проживает в европейской части, то 
все сервера будут располагаться в Московской области. Причем каждый из серверов
тройки master-slave-slave должны располагаться на 3-х разных хостингах. Также балансировщики,
application-сервера должны быть равномерно распределены по 3-м хостингам.


Устойчивость к сбоям:
--------------------------------
Так как сервера распределены по 3-м различным хостингам, то неполадки в одном из 
хостингов не завалят сервис - он останется доступным.

При неполадках в конкретных серверах сервис останется доступным:
1. Если application-сервер вышел из строя, то балансировщики перестанут перенаправлять к нему
запросы - сервис останется доступным.
2. Если балансировщик выйдет из строя, то это обнаружится при heart-beat-проверке (которая 
расположена на DNS-серверах), что повлечет удаление их адресных записей из DNS-таблиц - сервис останется доступным.
3. Если выйдет из строя DNS сервер, то пользователь после неудачного резолвинга адреса 
будет идти на второй DNS сервер - сервис останется доступным.
3. Если master выйдет из строя, его заменит slave, кластер сменит конфигурацию, после чего 
запросы на запись будут приходить вновь объявленному master - сервис останется доступным.
4. Если slave выйдет из строя, то есть ещё один slave, который начнет принимать все запросы 
на чтание - сервис останется доступным.

При всех этих раскладах у людей будет время на устранение недостатков сервиса и 
если им правильно воспользоваться, то сервис останется доступным.










