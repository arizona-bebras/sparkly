<script lang="ts">
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import { page } from "$app/state";
	import SparklesIcon from "@lucide/svelte/icons/sparkles";
	import LayoutDashboardIcon from "@lucide/svelte/icons/layout-dashboard";
	import CpuIcon from "@lucide/svelte/icons/cpu";
	import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
	import BoxesIcon from "@lucide/svelte/icons/boxes";
	import SettingsIcon from "@lucide/svelte/icons/settings";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";

	const parkId = $derived(page.params.park_id ?? "");
	const currentPath = $derived(page.url.pathname);

	const basePath = $derived(`/control-panel/${parkId}`);

	const mainNavItems = $derived([
		{
			title: "Панель управления",
			url: basePath,
			icon: LayoutDashboardIcon,
			isActive: currentPath === basePath || currentPath === `${basePath}/`,
		},
		{
			title: "Устройства",
			url: `${basePath}/devices`,
			icon: CpuIcon,
			isActive: currentPath.startsWith(`${basePath}/devices`),
		},
		{
			title: "Проблемы",
			url: `${basePath}/issues`,
			icon: CircleAlertIcon,
			isActive: currentPath.startsWith(`${basePath}/issues`),
		},
		{
			title: "Группы и релизы",
			url: `${basePath}/groups`,
			icon: BoxesIcon,
			isActive: currentPath.startsWith(`${basePath}/groups`),
		},
	]);

	const systemNavItems = $derived([
		{
			title: "Настройки парка",
			url: `${basePath}/settings`,
			icon: SettingsIcon,
			isActive: currentPath.startsWith(`${basePath}/settings`),
		},
	]);
</script>

<Sidebar.Root collapsible="icon">
	<Sidebar.Header class="p-3 border-b border-sidebar-border">
		<div class="flex items-center gap-2 font-semibold">
			<div class="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
				<SparklesIcon class="size-4" />
			</div>
			<div class="flex flex-col gap-0.5 leading-none group-data-[collapsible=icon]:hidden">
				<span class="text-base font-bold tracking-tight">Sparkly</span>
				<span class="text-xs text-muted-foreground font-mono">Парк #{parkId}</span>
			</div>
		</div>
	</Sidebar.Header>

	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupLabel>Навигация</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					{#each mainNavItems as item (item.title)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton tooltipContent={item.title} isActive={item.isActive}>
								{#snippet child({ props })}
									<a href={item.url} {...props}>
										<item.icon />
										<span>{item.title}</span>
									</a>
								{/snippet}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		<Sidebar.Group class="mt-auto">
			<Sidebar.GroupLabel>Система</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					{#each systemNavItems as item (item.title)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton tooltipContent={item.title} isActive={item.isActive}>
								{#snippet child({ props })}
									<a href={item.url} {...props}>
										<item.icon />
										<span>{item.title}</span>
									</a>
								{/snippet}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>

	<Sidebar.Footer class="p-3 border-t border-sidebar-border">
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton tooltipContent="Ко всем паркам">
					{#snippet child({ props })}
						<a href="/" {...props}>
							<ArrowLeftIcon />
							<span>Ко всем паркам</span>
						</a>
					{/snippet}
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
