# Курсовой проект "NotWhatsApp" #

![logo](https://github.com/ApTyp5/messanger.highload.techno/blob/main/notwatsup.jpg)

***

## Планируемая нагрузка

|Метрика|Комментарий|Значение|
|-------|-----------|:--------:|
|Аудитория|[(ayдитория телеграмма 4 года назад)](https://telegram.org/blog/100-million) / 2| 50 млн человек|
|средний request per day | [rpd 100 миллионов пользователей](https://telegram.org/blog/15-billion) - 15 млрд | 7.5 * 10^9 rpd|
|средний request per hour| rpd / 24 | 312 500 000 rph|
|средний rpm | rph/60 | 5 208 333 rpm|
|средний rps |rpm/60| 86 806 rps|

Учитывая, что в худшем случае каждому сообщению будет соответствовать запрос полученных сообщений пользователя то: 

1. Отправка сообщений займет _86 806 rps_
2. Запрос полученных сообщений _86 806 rps_

Возможны локальные пики нагрузок, под которые следует выделять в 2 раза больше ёмкости сервиса: 

1. 2 * 100 * 10^6 = __173 611 rps__ на запись
1. 2 * 100 * 10^6 = __173 611 rps__ на чтение

Также полезно узнать количество _уникальных пользователей в пиковые часы_. Будем считать, что их колчичество
в час равно 20%, а именно _50 000 000 * 20% / 100% = 10 млн чел/час_. Тогда в минуту в среднем 
чатиться начинает _10 * 10^6 / 60 = 166 667 чел/мин_. Пусть в локальный пик к нам может прийти в 2 раза больше,
то есть _166 667 * 2 = **333 334 макс. чел/мин**_, или _333 334 / 60 = **2778 макс. чел/сек**_.

Так как [средняя пользовательская сессия в watsapp](https://www.content-review.com/articles/46006/#:~:text=%D0%92%20WhatsApp%20%D1%81%D1%80%D0%B5%D0%B4%D0%BD%D1%8F%D1%8F%20%D0%B4%D0%BB%D0%B8%D1%82%D0%B5%D0%BB%D1%8C%D0%BD%D0%BE%D1%81%D1%82%D1%8C%20%D1%81%D0%B5%D1%81%D1%81%D0%B8%D0%B8,%2C%20%D0%B2%20%D0%B4%D0%B5%D0%BA%D0%B0%D0%B1%D1%80%D0%B5%20%E2%80%93%2013%20%D0%BC%D0%B8%D0%BD%D1%83%D1%82.)
длится 12 минут, то максимум в нашем сервисе будет _12 * 333 334 = **2 * 10^6  уник. подключений**_.

***

## Концептуальное описание
В MVP сервиса будет входить только отправка символьных сообщений, которя будет производиться следующим образом: 
1. К к каждому пользователю формируется очередь сообщений;
2. Пользователь может в любой момент получить сообщения, находящиеся в своей очереди;
3. История сообщений хранится на клиентах.

***

## Особенности функционирования
Иллюстрация всего, что будет описано в этом разделе
![Функционирование системы](https://github.com/ApTyp5/messanger.highload.techno/blob/main/alll.jpg)

### Логическая схема БД
![Схема бд](https://github.com/ApTyp5/messanger.highload.techno/blob/main/schem.jpg)

### Физическая особенность хранения очередей
Очереди сообщений хранятся следующим образом:
1. На каждого пользователя выделяется список, ключом которого является id пользователя.
2. В списке хранятся строковые записи.
3. Каждое отдельное сообщение состоит из 3-х полей: id автора, timestamp, text.
4. После считывания очередь удаляется.

### Шардинг / репликация
 * Шардинг будет производиться по полю user_id;
 * В кластере Redis доступно 16 383 _hash slots_, которые будут находиться по формуле "_CRC16(user_id) mod 16384_";
 * Каждый шард будет хранить подмножество _hash slots_, которое можно менять в процессе работы ситемы;
 * Для добавления новой ноды в кластер нужно:
   1. Добавить ноду в кластер;
   2. Если указанный _hash slot_ __не__ существуют на других нодах кластера, то указать, за какие _hash slots_ она ответственна;
   3. Иначе _hash slot_ надо [мигрировать](https://redis.io/topics/cluster-spec#moved-redirection)
   в добавленную ноду, причем во время миграции слот остаётся доступен;
 * Для удаления ноды из кластера надо [мигрировать](https://redis.io/topics/cluster-spec#moved-redirection) все 
 _hash slots_ с этой ноды на другие ноды в кластере, после чего со спокойной душой удалить ноду из кластера.
 * Каждого мастера будут страховать 2 слейва.
 * Операции чтения обрабатываются слейвами, записи - мастерами.

### Протоколы
Обмен трафиком между клиентом и сервисом будет происходить через wss, внутри сервиса - ws, а также внутренний 
протокол бд.

### Терминация SSL
Терминация SSL осуществяется на балансировщиках, обмен трафиком внутри сервиса 
осуществляется по незащищённому каналу.

### Балансировка
В проекте используется DNS балансировка. DNS-сервера отдают virtual-ip балансировщиков по алгоритму round-robin.

Virtual-ip, который получает пользователь, ведёт его на один из redundancy group, каждая из которых состоит из 2-х L7 балансировщиков,
связанных по CARP протоколу. Один из балансировщиков, сотоящий в redundancy group, по алгоритму round-robin будет 
проксировать трафик на один из app-серверов.

***

## Выбор технологий
* СУБД ~---~ Redis, так как он умеет работать со списками, а также имеет 
встроенные шардинг и репликацию;
* L7 балансировщик ~---~ nginx, так как он достаточно эффективен и лекго настраиваем;
* ЯП ~---~ golang, так как он имеет встроенную многопоточность, большое сообщество и его легко изучать;
* Фреймворк ~---~ echo, так как он популярный, простой, с хорошей документацией.

***

## Ресурсные затраты
__Критическим ресурсом__ для проектируемого сервиса является __cpu__. После [расчета _cpu_](#расчет-количества-ядер)
будет проведен [расчет требуемой __памяти__](#расчет-требуемой-памяти). Затем будет провещён расчет потребного оборудования

### Расчет количества ядер
#### Замеры времени вставки одного сообщения
![Замеры времени вставки одного сообщения](https://github.com/ApTyp5/messanger.highload.techno/blob/main/insert-bench.png)

Длина сообщения была равна 35 русских символов c пробелами, что соответствует 
[средней длине сообщения человека в возрасте 25 лет](https://crushhapp.com/blog/k-wrap-it-up-mom).
Округлим до __30 000 rps__.

#### Замеры времени получения одной очереди сообщений
![Замеры времени получения одной очереди сообщений](https://github.com/ApTyp5/messanger.highload.techno/blob/main/read-bench.png)

В очереди сообщений при замерах находится 20 писем.
Округлим до __30 000 rps__.


#### Эффективность Appliction серверов на golang:
Судя по [бенчмарку](https://github.com/smallnest/go-web-framework-benchmark), прокидывание запросов к СУБД из golang и последующий возврат ответа
даёт в среднем результаты __30 000 rps__.



#### Балансировщики нагрузки:
Исходя из [официальных бенчмарков](https://www.nginx.com/blog/nginx-websockets-performance/), 
nginx на 1 ядре и 1ГБ спокойно держит __50 000 подключений__, причем во время бенчмарка по этим подключениям
посылались сообщения размером от 10 до 4096 байт с шагом от 0.1 до 10 секунд, что досаточно точно 
имитирует поведение пользователя мессенджера.


***

## Требуемое оборудование
### Требуемое количество ядер
|Категория сервера|Вычисления|Количество ядер|
|---------|----------|:---------------:|
|Мастер-ноды|(173 611 rps) / (30 * 10^3 rps) * 5-кратный запас|30|
|Слейв-ноды|<количество ядер на master-node> * 2|60|
|App-серверы|(347 222 224 rps) / (30 * 10^3 rps) * 5-кратный запас|60|
|Balancers(подключения)|(2 * 10^6 одновременных подключений) / (50 * 10^3 подключений на ядро) + 8 (на ssl для 2778 подкл/сек)| 48 |

На терминацию ssl от _2778 макс. чел/сек_ L7 балансировщикам понадобилось, исходя из 
[официальных берчмарков](https://www.nginx.com/blog/testing-the-performance-of-nginx-and-nginx-plus-web-servers/), ещё дополнительно 8 ядер.

Так как DNS многократно кэшируется, то высоконагруженной подсистему DNS назвать нельзя. В нашем
случае DNS сервера будут обеспечивать мониторинг доступности балансировщиков (подробнее 
в обеспечении доступности). Однако для этого
также не нужно много ресурсов, поэтому для этих серверов подойдут 4-хядерные
8-гигабайтные машины.

***

### Расчет требуемой памяти
#### Сколько сообщений может максимум находиться в очереди чтения?
Так как MVP не подразумевает чаты, то 1000 сообщений на пользователя будет более чем достаточно
и этого хватит даже на "мёртвых душ".

#### Сколько памяти занимают 1000 средних сообщений?
![Сколько памяти занимают 1000 средних сообщений?](https://github.com/ApTyp5/messanger.highload.techno/blob/main/1000memory-usage.png)

Итак, максимальное число сообщений на одного пользователя займёт _25 631 байт = 26КБ_.

#### Сколько памяти нужно выделить на 1 ядро?
Так как 1 ядро спокойно может обслуживать 30 000 rps, то есть очереди 30 000 пользователей,
значит под этих пользователей понадобится _30 000 * 26КБ = 780 000КБ = 762 МБ_, а вместе с 
экземпляром redis (1МБ) объем занимаемой памяти будет равен __763MБ/ядро = 0.74ГБ на ядро__.

 
### Расчет памяти для долго ожидающих сообщений

Пусть средняя длина очереди сообщений равна 20. Тогда эти сообщения занимают _656 байт_. 

![Сколько памяти занимают 20 средних сообщений?](https://github.com/ApTyp5/messanger.highload.techno/blob/main/20memory-usage.png)

Тогда в 1 Гб памяти вместится  _1024^3 / 656 = 1 636 801_ средних пользователей. 

Пусть максимум средних пользователей, которые не читают сообщения ежечасно - четверть всех пользователей = 
_50 000 000 / 4 = 12 500 000_.

На этих средних пользователей надо зарезервировать _12 500 000 / 1 636 801 = 7.63 ~ 8Гб_ памяти.



***


## Расчет потребного оборудования

Для серверов будут использованы стандартные 32-х ядерные сервера (48 для балансировщика), память будет посчитана позднее.

#### Расчет количества серверов
|Категория сервера|Комментарий|Количество серверов|
|---------|----------|:---------------:|
|Мастер-ноды|30 / 32| 1|
|Слейв-ноды|1 * 2|2|
|App-серверы| 60 / 32| 2|
|Balancers L7| 1 (+ 1 доп. для отказоустойчивости) | 2 |
|DNS Balancres| 1 (+ 1 доп. для отказоуйстойчивости) | 2 |

#### Расчет количества памяти на конкретном типе серверов
|Категория сервера|Комментарий|Количество плашек памяти (16Gb DDR4, 4Gb DDR3)| Всего памяти на сервер (ГБ)|
|---------|----------|:---------------:|:-:|
|Мастер-ноды|32 ядра * 0.74ГБ/ядро + 8Гб на долгочитающих польователей = 31.68 ГБ| 2 DDR4| 32
|Слейв-ноды|=//=|2 DDR4|32| 
|App-серверы|16 ГБ на горутины| 1 DDR4| 16| 
|Balancers L7| 1 ГБ на ядро (48 ядер) | 3 DDR4 | 48|
|DNS Balancres| 8 ГБ | 2 DDR3 | 16|

### Итог оснащения сервиса:
|Категория сервера|CPU(cores)|RAM(GB)| Количество|
|---------|----------|:---------------:|:-:|
|Мастер-ноды|32|32| 1|
|Слейв-ноды|32|32| 2|
|App-серверы|32|16| 2|
|Balancers|48|48| 2|
|DNS|4|8|2|
|Запасные(мастер-слейв)|32|32|3|
|Запасные(app)|32|32|1|

Запасные сервера пригодятся для быстрого реагирования на непредвиденные 
ситуации.



***

## Инфраструктура
Так как большая часть населения России проживает в европейской части, то 
все сервера будут располагаться в Московской области. Причем каждый из серверов
тройки master-slave-slave должны располагаться на 3-х разных хостингах. Также балансировщики,
application-сервера должны находиться по 2-м хостингам.

***

## Отказоустойчивость

Так как сервера распределены по 3-м различным хостингам, то неполадки в одном из 
хостингов не завалят сервис - он останется доступным.

При неполадках в конкретных серверах сервис останется доступным:
1. Если application-сервер вышел из строя, то балансировщики перестанут перенаправлять к нему
запросы - сервис останется доступным.
2. На случай падения балансировщика его заменит 1 запасной, который соединён в redunancy group с ним по протоколу CARP. 
3. Если выйдет из строя DNS сервер, то пользователь после неудачного резолвинга адреса 
будет направлен на второй DNS сервер (DNS сервера объеденены по протоколу CARP) - сервис останется доступным.
3. Если master выйдет из строя, его заменит slave, кластер сменит конфигурацию, после чего 
запросы на запись будут приходить вновь объявленному master - сервис останется доступным.
4. Если slave выйдет из строя, то есть ещё один slave, который начнет принимать все запросы 
на чтание - сервис останется доступным.
5. 3 запасные сервера также будут подключены к кластеру как slave-ноды, чтобы при выходе из
строя более одного сервера из кластера сервер оказался доступным.
6. 1 запасной сервер будет страховать app-сервера на случай выхода из строя одного из них. 

При всех этих раскладах у людей будет время на устранение недостатков сервиса и 
если они им правильно воспользоваться, то сервис останется доступным.










