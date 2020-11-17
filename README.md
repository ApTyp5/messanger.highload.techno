Курсовой проект "мессенджер"
==========

Задача: 
---------
1. Определить примерное количество пользователей.
2. Определить планируемые (пиковые) нагрузки на сервис.
3. Составить логическую схему базы данных для MVP


Аудитория: 
---------
50 млн человек по России [(половина пользователей телеграмма 4 года назад)](https://telegram.org/blog/100-million)

Планируемые нагрузки:
----------------
[100 миллионов пользователей](https://telegram.org/blog/100-million) были способны создавать нагрузку в 
[15 биллионов запросов](https://telegram.org/blog/15-billion) в сутки

Следовательно для 50 млн пользователей нагрузка будет равна 
(15 * 10^9) / 2 = __7.5 * 10^9 (request per day)__

Но нагрузка распределена не равномерно, в соответствии со [статистикой](https://popsters.ru/blog/post/aktivnost-auditorii-v-socialnyh-setyah-issledovanie-2019) 
в пиковый час телеграм обрабаывает 4,5 % дневного трафика.

Следовательно, в пик нагрузки сервису приходится обрабаывать 
(7.5 * 10^9 * 4,5%) / 100% = __3 375 * 10^5 (rph)__

Предположив,что в течение часа/минуты нагрузка распределяется равномерно, расчитаем более точную нагрузку на сервис: 
```
  3 375 * 10^5 (request per hour) =
  = 56,25 * 10^5 (rpm) =
  = 93,75 * 10^3 (rps)
```


Учитывая, что для MVP релаизация чатов не потребуется, то траффик можно разделить на 2 равных составляюих: 

1. Отправка сообщений _(93,75 / 2 * 10^3) = 46,875 * 10^3 ~= 47 * 10^3_
2. Запрос на уведомления ~= _47 * 10^3_


Схема бд:
--------------------
![Схема бд](https://github.com/ApTyp5/messanger.highload.techno/blob/main/scheme.jpg)



Выбор конкретной СУБД
-----------------------------
В качестве СУБД выбрана MongoDB благодаря следующим характеристикам:
1. Репликация из коробки;
2. Мастабируемость из коробки;
3. Большое количество хорошей документации;
3. Низкий порог входа ([легко настраивать](https://github.com/ApTyp5/messanger.highload.techno/blob/main/MongoDB_Architecture_Guide.pdf)).



Идея функционирования сервиса:
--------------------
1. Обращения к таблицам пользователей и чатов происходят не часто, поэтому подробно рассматриваться не будут.
2. Основная функциональность (отправка сообщений и запрос на уведомления) будут реализовываться при помощи таблицы сообщений.
3. Отправка сообщений есть вставка новой записи в таблицу сообщений.
4. Запрос уведомлений есть считывание сообщений чата chat_id начиная со времени timestamp (последняя метрика хранится на клиенте).



Взаимодействие всех частей в целом:
--------------------------------
эта картинка должна быть в конце как итог, но приведена сейчас, чтобы было легче понимать, о чем говорится дальше
![Схема бд](https://github.com/ApTyp5/messanger.highload.techno/blob/main/all.jpg)


Шардинг
----------------------------
Шардинг в MongoDB состоит из:
1. Серверов конфигураций (их должно быть 3 штуки, подробно рассматриваться не будут);
2. Шардов (подробости рассмотрены далее);
3. Клиентских демонов (по демону на каждый app-server, он действует в качестве роутера между шардами);

Шардинг будет производиться по полю chat_id, количество шардов будет вычислено далее. 
Данные в шардах можно разделять на чанки (например по 10 чатов на 1 чанк). Они при неравномерном
занимаемом пространстве автоматически балансироваться между шардами.



Анализ выдерживаемой нагрузки при отправке сообщений (то есть при вставке нового документа в базу):
--------------------------
1. Изначально была замерена нагрузка записи новых сообщений в изначально пустую СУБД (_9000qps_)
2. Но ведь в бою база не будет пустая, в худжей ситуации: _9000rps * 60s * 60m * 24h = 939 600 000_ - столько
сообщений должна вмещать база данных в идеальном случае.
3. На деле деградация началась на 12 000 000 сообщений, то есть на 10 000 000 уже следует менять Primary node с одной
из реплик, на которой предварительно были сжаты и выгружены все прочитанные сообщения.
4. В связи с предыдущим пунктом операции чтения с application серверов следует посылать на реплики.
5. Итоговое количество primaryNode (а именно количество шардов) - 11 штук + 1 запасной = 12 (чтобы покрыть _93,75 * 10^3rps).

*N.B. Запись в одну таблицу не может производиться более, чем одним потоком* 



Подсчет выдерживаемой нагрузки при получении уведомлений (то есть при вставке нового документа в базу):
---------------------------------------------------
1. При заполнености базы в 10 000 000 сообщений (максимум, который мы допускаем), выборка из 
уведомлений по одному чату начиная с такого-то времени заняла _4000 rps_.
2. Но так как все ресурсы Primary не используются (для записи используется только одно ядро), то часть нагрузки на выдачу уведомлений он сможет взять на себя (4000 rps).
3. Итоговое количество реплик для каждого Primary: 4 штуки - 1 для выдерживания прямой нагрузки на чтение, 1 для выгрузки данных
(см. пункт 3 предыдущего раздела) и 1 на случай выхода из строя одной из нод.


*P.S. все замеры проводились на 2-хядерном процессоре (4 логических) и 8 Гб оперативки, что в случае 
несоответствия нагрузок в бою нагрузкам при тестировании даёт большое поле для быстрого вертикального масштабирования.*


Итог оснащения СУБД:
------------------------------------
1. Для обеспечения обслуживания 93,75 * 10^3 qps потребуется (12 * 4{шарды}) + (3{конфигурационные}) = 48 серверов (2 ядра, 8+ Гб оперативки)


Appliction серверы:
------------------------------------
Судя по [исследованиям](https://github.com/smallnest/go-web-framework-benchmark), прокидывание запросов к СУБД из golang(обоснование позже) и последующий возврат ответа 
даёт в среднем результаты 30 000 rps. Следовательно нам для нашей задачи (93,75 * 10^3rps) понадобится 5 application серверов (1 запасной).



Балансировщики нагрузки:
--------------------------------------
Исходя из [исследований](https://github.com/NickMRamirez/Proxy-Benchmarks), nginx (обоснование позже) проксирует запросы 
на сервера с ёмкостью 30000rps. Следовательно нам для нашей задачи (93,75 * 10^3rps) понадобится 5 балансировщиков нагрузки (1 запасной).
Также для DNS балансировки желательно иметь свой DNS сервер (1) и запасной ему, который переодически опрашивает собрата о состоянии и 
при надобности заменяет его (1).

Особенности балансировки: DNS отдают адреса балансировщиков по round-robin с ttl 5min. таким образом при выходе из строя одного балансировщика
достаточно будет удалить его из записей DNS.


Итог оснащения Сервиса:
-------------------------------
1. СУБД, шарды. 2 ядра 16 Гб памяти. 45 серверов.
2. СУБД, конфиг. 2 ядра 8 Гб памяти. 3 сервера.
2. App. 4 ядра 8 Гб памяти. 5 серверов.
3. Balance. 4 ядра 8 Гб памяти. 5 серверов.
4. DNS сервер. 4 ядра 8 Гб памяти. 2 сервера.


Выбор прочих технологий:
-------------------------------
1. ЯП, фреймворк: golang, echo. Преимущества: быстрая разработка, встроенная конкурентность, большое сообщество, самый производительный фрейсворк.
2. Протоколы взаимодействия: https при установлении соединения, после wss, так как это наиболее удобный протокол для поддержки мессенджера.
3. Веб-сервер: nginx. Так как он обладает высокой проиводительностью и легко настраиваем.


Облачный провайдер: 
--------------------------------
Google Cloud, так как позволяет быстро и гибко настроить весь сервис, ближайший датацентр построен по стандарту tier IV. Также даёт возможность 
быстро расшириться под нагрузку при всплеске нагрузки.


Расположение серверов, устойчивость к сбоям:
--------------------------------
сервера будут равномерно распределены по зонам a,b,c. Причем также равномерно должны быть распределены сервера в пределах шарда.
Таким образом при выходе из строя одной зоны сервис не выйдет из строя - в шардах будут выбраны новые primary ноды, 
вышедшие из строя балансировщики будут удалены из DNS записей, упавшие application сервера будут игнорироваться балансировщиками.
Поэтому сервис останется доступным.









