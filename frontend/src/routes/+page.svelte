<script>
import { createQuery } from "@tanstack/svelte-query";
import { client } from "$proto";

const query = createQuery(() => ({
	queryKey: ["say"],
	queryFn: async () =>
		await client.say({
			sentence: "I feel happy. Tell me about yourself",
		}),
}));
</script>  


<svelte:head>
    <title>Sparkly</title>
</svelte:head>

<main>
    <h1>Sparkly</h1>
</main>
