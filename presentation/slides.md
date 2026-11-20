---
theme: apple-basic
title: Sparkly — мониторинг парка IoT
info: Презентация проекта Sparkly. Демонстрационные данные и гипотезы эффекта отмечены отдельно.
author: Sparkly
favicon: "data:,"
colorSchema: dark
aspectRatio: 16/9
canvasWidth: 1280
fonts:
  sans: IBM Plex Sans
  mono: JetBrains Mono
  weights: "400,500,600,700"
  provider: none
transition: fade-out
defaults:
  layout: default
drawings:
  enabled: false
class: sp-cover
---

<DeckHeader section="Мониторинг устройств" context="IoT / Телеметрия" />

<div class="sp-hero">
  <div class="sp-hero-copy">
    <h1>Sparkly</h1>
    <p class="sp-hero-lead">Мониторинг и телеметрия парка IoT</p>
    <p class="sp-hero-detail">Метрики, логи и инциденты контроллеров в одном контуре.</p>
    <StatusBadge label="ИЗОЛИРОВАННЫЙ КОНТУР" tone="accent" />
  </div>
  <div class="sp-cover-visual">
    <div class="sp-cover-readout" aria-hidden="true">
      <div class="sp-readout-head sp-mono"><span>ТЕЛЕМЕТРИЯ / ESP32</span><span class="sp-accent">●</span></div>
      <svg viewBox="0 0 500 180" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path class="sp-sv-grid" d="M0 30H500M0 70H500M0 110H500M0 150H500M25 0V180M100 0V180M175 0V180M250 0V180M325 0V180M400 0V180M475 0V180" />
        <path class="sp-sv-signal" d="M0 126h39l15-5h46l14 6h51l18-4h57l12 3h57l17-6h55l10 4h47l13-2h49" />
        <path class="sp-sv-accent" d="M0 94h36l17 4h49l15-12h46l20 4h26l18-12h24l20 8h22l22-16 14-41 17 67h42l15-11h44l15 5h38" />
        <path class="sp-sv-guide" d="M329 0v180" />
        <circle cx="329" cy="29" r="5" fill="var(--sp-accent)" />
      </svg>
      <div class="sp-readout-foot sp-mono"><span>01 / МЕТРИКИ</span><span>02 / ЛОГИ</span><span>03 / СОБЫТИЯ</span></div>
    </div>
    <p class="sp-mono sp-meta">Иллюстрация потока телеметрии</p>
  </div>
</div>

<DeckFooter note="Платформа наблюдаемости для IoT" />

---

title: Причины сбоев не видны
---

<DeckHeader section="Проблема" />

<div class="sp-problem">
  <div>
    <h1 class="sp-statement">Устройства работают в поле.<br><span class="sp-accent">Причины сбоев<br>не видны.</span></h1>
    <p>Без телеметрии инженер узнаёт о сбое, но не видит, что происходило до него.</p>
  </div>
  <ol class="sp-issues">
    <li><span class="sp-mono sp-meta">01</span><div><h2>Данные разрознены</h2><p>Логи и показания контроллера приходится собирать вручную.</p></div></li>
    <li><span class="sp-mono sp-meta">02</span><div><h2>Теряется контекст сбоя</h2><p>Трудно связать ошибку с устройством, временем и перезагрузкой.</p></div></li>
    <li><span class="sp-mono sp-meta">03</span><div><h2>Диагностика требует выезда</h2><p>Иногда к контроллеру приходится ехать, чтобы начать разбор.</p></div></li>
  </ol>
</div>

<DeckFooter note="Эксплуатация распределённого парка" />

---

<DeckHeader section="Решение" />

# От сигнала устройства к расследованию

<p class="sp-intro">Sparkly связывает телеметрию с конкретным устройством и моментом сбоя.</p>

<ol class="sp-route">
  <li><div class="sp-route-mark"><span class="sp-mono">01</span><lucide:cpu /></div><h2>Сбор</h2><p>Агент записывает метрики, логи и факты перезагрузок.</p></li>
  <li><div class="sp-route-mark"><span class="sp-mono">02</span><lucide:radio-tower /></div><h2>Передача</h2><p>Устройство отправляет Protobuf по защищённому каналу.</p></li>
  <li><div class="sp-route-mark"><span class="sp-mono">03</span><lucide:database /></div><h2>Хранение</h2><p>Сервис направляет логи и метрики в разные хранилища.</p></li>
  <li><div class="sp-route-mark"><span class="sp-mono">04</span><lucide:search /></div><h2>Расследование</h2><p>Оператор открывает инцидент и сверяет сигналы перед сбоем.</p></li>
</ol>

<DeckFooter note="Единый маршрут диагностики" />

---

<DeckHeader section="Аудитория" />

# Для кого создаётся Sparkly

<p class="sp-intro">Три роли, которым нужны одни и те же сигналы, но для разных задач.</p>

<div class="sp-audience">
  <article class="sp-card sp-role">
    <lucide:cpu />
    <h2>Команды прошивок</h2>
    <p>Разбирают сбои контроллеров по логам, метрикам и фактам перезагрузок.</p>
    <span class="sp-mono sp-meta">01 / ВСТРОЕННОЕ ПО</span>
  </article>
  <article class="sp-card sp-role">
    <lucide:server />
    <h2>Платформенные инженеры</h2>
    <p>Подключают устройства и разворачивают сбор телеметрии в своём контуре.</p>
    <span class="sp-mono sp-meta">02 / ИНФРАСТРУКТУРА</span>
  </article>
  <article class="sp-card sp-role">
    <lucide:panels-top-left />
    <h2>Операторы парка</h2>
    <p>Проверяют доступность устройств и открывают события, связанные со сбоем.</p>
    <span class="sp-mono sp-meta">03 / ЭКСПЛУАТАЦИЯ</span>
  </article>
</div>

<DeckFooter note="Разработка · инфраструктура · эксплуатация" />

---

<DeckHeader section="Границы релиза" />

# Что входит в первый релиз

<p class="sp-intro">Первый релиз — для наблюдения за парком и диагностики сбоев.</p>

<div class="sp-scope">
  <article class="sp-card sp-scope-now">
    <StatusBadge label="MVP" tone="accent" />
    <h2>Мониторинг и инциденты</h2>
    <ul>
      <li>Регистрация устройств по токену группы</li>
      <li>Сбор метрик, текстовых логов и фактов перезагрузок</li>
      <li>Приём и маршрутизация телеметрии</li>
      <li>Устройства, группы, логи, графики и статусы доступности</li>
      <li>Фиксация критических ошибок как инцидентов</li>
    </ul>
  </article>
  <article class="sp-card sp-scope-later">
    <StatusBadge label="ПОЗЖЕ" />
    <h2>Управление и глубокая диагностика</h2>
    <ul>
      <li>OTA и безопасный откат</li>
      <li>Coredump, слепки RAM и разбор debug symbols</li>
      <li>Двустороннее командное управление</li>
      <li>Расширенная ролевая модель</li>
      <li>Предиктивная аналитика</li>
    </ul>
  </article>
</div>

<DeckFooter note="План релиза · функции будущих этапов не входят в MVP" />

---

<DeckHeader section="Архитектура" />

# Как устроен Sparkly

<p class="sp-intro">Устройство отправляет Protobuf по защищённому каналу.</p>

<ArchitectureMap />

<DeckFooter note="Целевая схема первого релиза" />

---

<DeckHeader section="Телеметрия" />

# Что видит оператор

<p class="sp-intro">График и журнал событий помогают восстановить ход сбоя.</p>

<div class="sp-console">
  <div class="sp-console-bar">
    <span class="sp-mono">ПАРК <span class="sp-console-sep">/</span> ГРУППА A <span class="sp-console-sep">/</span> <strong>ESP32-017</strong></span>
    <StatusBadge label="ONLINE" tone="success" />
  </div>
  <div class="sp-console-body">
    <TelemetryChart />
    <LogPanel />
  </div>
</div>

<DeckFooter note="Макет интерфейса · демонстрационные данные" />

---

<DeckHeader section="Инциденты" />

# От критического события к инциденту

<p class="sp-intro">После перезагрузки Sparkly фиксирует сбой; причину проверяет инженер.</p>

<div class="sp-timeline-heading"><span class="sp-mono">ESP32-017</span><span class="sp-meta">Демонстрационный сценарий</span></div>
<ol class="sp-timeline">
  <li><time class="sp-mono">12:41:08</time><h2>Предупреждение</h2><p>Агент сообщает о низком напряжении питания.</p></li>
  <li><time class="sp-mono">12:41:09</time><h2>Перезагрузка</h2><p>Агент передаёт факт внеплановой перезагрузки после запуска.</p></li>
  <li><span class="sp-mono sp-time">После приёма</span><h2>Инцидент</h2><p>Sparkly привязывает критическое событие к устройству.</p></li>
  <li><span class="sp-mono sp-time">При разборе</span><h2>Проверка причины</h2><p>Инженер сверяет события и проверяет гипотезу о сбое питания.</p></li>
</ol>

<DeckFooter note="Порядок событий помогает выдвинуть гипотезу, но не доказывает причину" />

---

<DeckHeader section="Экономический эффект" />

# Меньше времени на сбой, меньше выездов

<p class="sp-intro">Целевые диапазоны для проверки на пилоте.</p>

<div class="sp-metrics">
  <MetricBlock value="30–60%" label="Сокращение MTTR" detail="Среднее время восстановления: от фиксации сбоя до возобновления работы." />
  <MetricBlock value="50–70%" label="Снижение затрат на выездную диагностику" detail="Часть неисправностей можно диагностировать удалённо." />
</div>
<p class="sp-metric-note"><StatusBadge label="ГИПОТЕЗА" tone="accent" /> Подтверждённых результатов пилота пока нет.</p>

<DeckFooter note="Ожидаемые показатели, не результаты измерений" />

---

title: Контроль над парком начинается с видимости сбоев
---

<DeckHeader section="Итог" context="IoT / Телеметрия" />

<div class="sp-final">
  <div>
    <div class="sp-final-brand">Sparkly</div>
    <h1 class="sp-statement">Контроль над парком начинается с видимости сбоев</h1>
    <p>Видеть состояние устройств, восстанавливать ход сбоя и решать, нужен ли выезд.</p>
  </div>
  <ol class="sp-final-flow" aria-label="Путь от устройства к диагностике">
    <li><lucide:cpu /><div><span class="sp-mono sp-meta">01 / УСТРОЙСТВО</span><strong>Метрики и логи</strong></div></li>
    <li><lucide:activity /><div><span class="sp-mono sp-meta">02 / СОБЫТИЕ</span><strong>Контекст сбоя</strong></div></li>
    <li><lucide:search /><div><span class="sp-mono sp-meta">03 / РАССЛЕДОВАНИЕ</span><strong>Удалённая диагностика</strong></div></li>
  </ol>
</div>

<DeckFooter note="Платформа наблюдаемости для IoT" />
