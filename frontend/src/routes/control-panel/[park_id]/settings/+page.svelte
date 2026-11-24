<script lang="ts">
  import { page } from '$app/state';
  import * as Accordion from '$lib/components/ui/accordion/index.js';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { RustIcon, CIcon } from '$lib/components/icons/index.js';
  import ParkCodeBlock from '$lib/components/entities/park/ParkCodeBlock.svelte';

  const initialOpen =
    page.url.searchParams.has('onboarding') ||
    page.url.searchParams.get('open') === 'onboarding' ||
    page.url.hash === '#onboarding';

  let accordionValue = $state<string>(initialOpen ? 'onboarding' : '');
  let selectedLang = $state<'rust' | 'c'>('rust');

  function switchLanguage(lang: 'rust' | 'c') {
    selectedLang = lang;
  }

  const examples = {
    rust: {
      file: 'main.rs',
      install: `# Установите CLI Sparkly.
cargo install sparkly-cli --locked

# В каталоге прошивки добавьте зависимости и настройте линкер.
sparkly init --chip esp32s3 --sdk-path /path/to/sparkly/agent`,
      configure: `use sparkly_agent::{counter, metric};
use sparkly_esp::{Agent, Config};

let config = Config::new(
    ENROLLMENT_URL,
    INGEST_URL,
    include_bytes!("sparkly-ca.pem"),
)
.with_enrollment_token(provisioning.enrollment_token());`,
      start: `// Используем Wi-Fi и постоянное хранилище прошивки.
let mut agent = Agent::start(config, &wifi, &mut credentials)?;
log::info!("Датчик запущен");`,
      sample: `// Вызывайте в цикле приложения.
let temperature = read_temperature_c();
metric!("sensor.temperature_c").set(temperature);
counter!("sensor.samples").increment(1);
if temperature > 40.0 {
    log::warn!("Высокая температура датчика: {temperature:.1} °C");
}

agent.poll().await?;`,
    },
    c: {
      file: 'main.c',
      install: `# Включите модуль в prj.conf: CONFIG_SPARKLY=y
west update sparkly`,
      configure: `#include <sparkly/zephyr.h>

const sparkly_config_t config = {
    .enrollment_url = ENROLLMENT_URL,
    .ingest_url = INGEST_URL,
    .ca_pem = sparkly_ca_pem,
    .enrollment_token = provisioning_enrollment_token(),
};`,
      start: `// Адаптер подключает сеть и постоянное хранилище Zephyr.
sparkly_agent_t *agent = NULL;
int status = sparkly_agent_start(&config, &zephyr_port, &agent);
if (status != SPARKLY_OK) {
    return status;
}
SPARKLY_LOG_INFO("Датчик запущен");`,
      sample: `// Вызывайте в цикле приложения.
double temperature = read_temperature_c();
SPARKLY_METRIC_SET("sensor.temperature_c", temperature);
SPARKLY_COUNTER_ADD("sensor.samples", 1);
if (temperature > 40.0) {
    SPARKLY_LOG_WARN("Температура датчика выше 40 °C");
}

status = sparkly_agent_poll(agent);
if (status != SPARKLY_OK) {
    return status;
}`,
    },
  };

  const example = $derived(examples[selectedLang]);
  const steps = $derived([
    {
      number: 1,
      title: 'Подключите библиотеку агента',
      file: 'Терминал',
      lang: 'bash' as const,
      code: example.install,
    },
    {
      number: 2,
      title: 'Передайте настройки подключения и токен парка',
      code: example.configure,
    },
    {
      number: 3,
      title: 'Запустите агент при старте прошивки',
      code: example.start,
    },
    {
      number: 4,
      title: 'Записывайте метрики и логи, обрабатывайте очередь',
      code: example.sample,
    },
  ]);
</script>

<svelte:head>
  <title>Настройки парка | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-8 pb-10">
  <div class="flex flex-col gap-1">
    <div class="flex items-center gap-2">
      <h1 class="text-2xl font-bold tracking-tight">Настройки парка</h1>
    </div>
    <p class="text-sm text-muted-foreground">
      Конфигурация параметров парка и руководство по интеграции устройств
    </p>
  </div>

  <Accordion.Root type="single" bind:value={accordionValue} class="bg-card shadow-xs">
    <Accordion.Item value="onboarding" class="border-none">
      <Accordion.Trigger class="px-5 py-4 hover:no-underline cursor-pointer">
        <div class="flex items-center gap-2.5">
          <span class="text-base font-semibold">Подключение устройства к парку</span>
          <Badge variant="secondary" class="font-normal text-xs">{steps.length} шага</Badge>
        </div>
      </Accordion.Trigger>

      <Accordion.Content class="px-5 pb-6 pt-1">
        <div class="flex pb-4 pt-1 border-b border-border/60 mb-6">
          <Tabs.Root
            value={selectedLang}
            onValueChange={(val) => {
              if (val === 'rust' || val === 'c') switchLanguage(val);
            }}
          >
            <Tabs.List class="grid grid-cols-2 w-40">
              <Tabs.Trigger
                value="rust"
                onclick={() => switchLanguage('rust')}
                class="flex items-center gap-1.5 font-medium cursor-pointer text-xs"
              >
                <RustIcon class="size-3.5 text-[#dea584]" />
                <span>Rust</span>
              </Tabs.Trigger>
              <Tabs.Trigger
                value="c"
                onclick={() => switchLanguage('c')}
                class="flex items-center gap-1.5 font-medium cursor-pointer text-xs"
              >
                <CIcon class="size-3.5" />
                <span>Zephyr</span>
              </Tabs.Trigger>
            </Tabs.List>
          </Tabs.Root>
        </div>

        <div class="flex flex-col">
          {#each steps as step, index (step.number)}
            <ParkCodeBlock
              number={step.number}
              title={step.title}
              file={step.file ?? example.file}
              code={step.code}
              lang={step.lang ?? selectedLang}
              isLast={index === steps.length - 1}
            />
          {/each}
        </div>
      </Accordion.Content>
    </Accordion.Item>
  </Accordion.Root>
</div>
