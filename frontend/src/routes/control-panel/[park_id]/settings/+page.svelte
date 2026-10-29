<script lang="ts">
	import { page } from "$app/state";
	import * as Accordion from "$lib/components/ui/accordion/index.js";
	import * as Tabs from "$lib/components/ui/tabs/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import { RustIcon, CIcon } from "$lib/components/icons/index.js";
	import ParkCodeBlock from "$lib/components/entities/park/ParkCodeBlock.svelte";

	const parkId = $derived(page.params.park_id);

	const initialOpen =
		page.url.searchParams.has("onboarding") ||
		page.url.searchParams.get("open") === "onboarding" ||
		page.url.hash === "#onboarding";

	let accordionValue = $state<string>(initialOpen ? "onboarding" : "");
	let selectedLang = $state<"rust" | "c">("rust");

	function switchLanguage(lang: "rust" | "c") {
		selectedLang = lang;
	}

	const codeByLang = {
		rust: {
			step1: `qniot::initialize("iFUCKyRMoM===");`,
			step1File: "main.rs",
			step2: `let token = qniot.getDeviceToken();\nflash.write(&token);`,
			step2File: "auth.rs",
			step3: `let token = flash.getToken();\nlogger.init(qniot.logger(token));\n\ninfo!("все работает");\ncounter!("reboots").increment(1);`,
			step3File: "telemetry.rs",
		},
		c: {
			step1: `qniot_initialize("iFUCKyRMoM===");`,
			step1File: "main.c",
			step2: `char* token = qniot_get_device_token();\nflash_write(token);`,
			step2File: "auth.c",
			step3: `char* token = flash_get_token();\nlogger_init(qniot_logger(token));\n\ninfo("все работает");\ncounter_increment("reboots", 1);`,
			step3File: "telemetry.c",
		},
	};

	const steps = $derived([
		{
			number: 1,
			title: "Инициализация прошивки токеном парка",
			code: codeByLang[selectedLang].step1,
			file: codeByLang[selectedLang].step1File,
		},
		{
			number: 2,
			title: "Регистрация устройства и сохранение токена",
			code: codeByLang[selectedLang].step2,
			file: codeByLang[selectedLang].step2File,
		},
		{
			number: 3,
			title: "Отправка метрик и логов в сервис",
			code: codeByLang[selectedLang].step3,
			file: codeByLang[selectedLang].step3File,
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
			<Badge variant="outline" class="font-mono text-xs">#{parkId}</Badge>
		</div>
		<p class="text-sm text-muted-foreground">
			Конфигурация параметров парка и руководство по интеграции устройств
		</p>
	</div>

	<Accordion.Root type="single" bind:value={accordionValue} class="bg-card shadow-xs">
		<Accordion.Item value="onboarding" class="border-none">
			<Accordion.Trigger class="px-5 py-4 hover:no-underline cursor-pointer">
				<div class="flex items-center gap-2.5">
					<span class="text-base font-semibold"
						>Подключение устройства к парку</span
					>
					<Badge variant="secondary" class="font-normal text-xs"
						>3 шага</Badge
					>
				</div>
			</Accordion.Trigger>

			<Accordion.Content class="px-5 pb-6 pt-1">
				<div class="flex pb-4 pt-1 border-b border-border/60 mb-6">
					<Tabs.Root
						value={selectedLang}
						onValueChange={(val) => {
							if (val === "rust" || val === "c")
								switchLanguage(val);
						}}
					>
						<Tabs.List class="grid grid-cols-2 w-40">
							<Tabs.Trigger
								value="rust"
								onclick={() => switchLanguage("rust")}
								class="flex items-center gap-1.5 font-medium cursor-pointer text-xs"
							>
								<RustIcon class="size-3.5 text-[#dea584]" />
								<span>Rust</span>
							</Tabs.Trigger>
							<Tabs.Trigger
								value="c"
								onclick={() => switchLanguage("c")}
								class="flex items-center gap-1.5 font-medium cursor-pointer text-xs"
							>
								<CIcon class="size-3.5" />
								<span>C</span>
							</Tabs.Trigger>
						</Tabs.List>
					</Tabs.Root>
				</div>

				<div class="flex flex-col">
					{#each steps as step, index (step.number)}
						<ParkCodeBlock
							number={step.number}
							title={step.title}
							file={step.file}
							code={step.code}
							lang={selectedLang}
							isLast={index === steps.length - 1}
						/>
					{/each}
				</div>
			</Accordion.Content>
		</Accordion.Item>
	</Accordion.Root>
</div>
