<script lang="ts">
  import '../app.css';
  import {
    QueryCache,
    MutationCache,
    QueryClient,
    QueryClientProvider,
    type DefaultError,
  } from '@tanstack/svelte-query';
  import { SvelteQueryDevtools } from '@tanstack/svelte-query-devtools';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';

  let { children } = $props();

  function handleAuthError(error: DefaultError) {
    if (error.message.includes('401') || error.message.includes('unauthenticated')) {
      if (!page.url.pathname.startsWith('/auth')) {
        goto('/auth/signin');
      }
    }
  }

  const queryClient = new QueryClient({
    queryCache: new QueryCache({
      onError: handleAuthError,
    }),
    mutationCache: new MutationCache({
      onError: handleAuthError,
    }),
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5,
        retry: (failureCount, error) => {
          if (error.message.includes('401') || error.message.includes('unauthenticated')) {
            return false;
          }
          return failureCount < 3;
        },
      },
    },
  });
</script>

<QueryClientProvider client={queryClient}>
  {@render children?.()}

  {#if import.meta.env.DEV}
    <SvelteQueryDevtools />
  {/if}
</QueryClientProvider>
