<script lang="ts">
  import * as Sidebar from '$lib/components/ui/sidebar/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { page } from '$app/state';
  import { createQuery } from '@tanstack/svelte-query';
  import { client } from '$proto';
  import {
    Sparkles,
    LayoutDashboard,
    Cpu,
    Boxes,
    Rocket,
    Package,
    CircleAlert,
    Settings,
    ArrowLeft,
  } from '@lucide/svelte';

  const parkId = $derived(page.params.park_id ?? '');
  const currentPath = $derived(page.url.pathname);

  const parkQuery = createQuery(() => ({
    queryKey: ['park', parkId],
    queryFn: () => client.park.getPark({ id: parkId }),
    enabled: !!parkId,
    staleTime: 1000 * 60 * 5,
  }));

  const park = $derived(parkQuery.data?.park);

  const basePath = $derived(`/control-panel/${parkId}`);

  const mainNavItems = $derived([
    {
      title: 'Панель управления',
      url: basePath,
      icon: LayoutDashboard,
      isActive: currentPath === basePath || currentPath === `${basePath}/`,
    },
    {
      title: 'Устройства',
      url: `${basePath}/devices`,
      icon: Cpu,
      isActive: currentPath.startsWith(`${basePath}/devices`),
    },
    {
      title: 'Группы',
      url: `${basePath}/groups`,
      icon: Boxes,
      isActive: currentPath.startsWith(`${basePath}/groups`),
    },
    {
      title: 'Релизы',
      url: `${basePath}/releases`,
      icon: Rocket,
      isActive: currentPath.startsWith(`${basePath}/releases`),
    },
    {
      title: 'Сборки',
      url: `${basePath}/builds`,
      icon: Package,
      isActive: currentPath.startsWith(`${basePath}/builds`),
    },
    {
      title: 'Проблемы',
      url: `${basePath}/issues`,
      icon: CircleAlert,
      isActive: currentPath.startsWith(`${basePath}/issues`),
    },
  ]);

  const systemNavItems = $derived([
    {
      title: 'Настройки парка',
      url: `${basePath}/settings`,
      icon: Settings,
      isActive: currentPath.startsWith(`${basePath}/settings`),
    },
  ]);
</script>

<Sidebar.Root collapsible="icon">
  <Sidebar.Header class="p-3 border-b border-sidebar-border">
    <div class="flex items-center gap-2.5 min-w-0">
      <div
        class="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground"
      >
        <Sparkles class="size-4" />
      </div>
      <div class="flex flex-col min-w-0 flex-1 leading-tight group-data-[collapsible=icon]:hidden">
        {#if parkQuery.isPending && !park}
          <Skeleton class="h-4 w-28 mb-1" />
          <Skeleton class="h-3 w-36" />
        {:else if park}
          <span class="text-sm font-bold tracking-tight truncate" title={park.name}
            >{park.name}</span
          >
          {#if park.description}
            <span class="text-xs text-muted-foreground truncate" title={park.description}
              >{park.description}</span
            >
          {/if}
        {:else}
          <span class="text-sm font-bold tracking-tight truncate">Sparkly</span>
          <span class="text-xs text-muted-foreground font-mono truncate">Парк #{parkId}</span>
        {/if}
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
              <ArrowLeft />
              <span>Ко всем паркам</span>
            </a>
          {/snippet}
        </Sidebar.MenuButton>
      </Sidebar.MenuItem>
    </Sidebar.Menu>
  </Sidebar.Footer>
  <Sidebar.Rail />
</Sidebar.Root>
