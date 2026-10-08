<script>
import { onMount } from "svelte";
import { client } from "$proto";

onMount(async () => {
	console.log("onMount");
	const res = await client.say({
		sentence: "I feel happy. Tell me about yourself",
	});
	console.log(res.sentence);
});
</script> 


<svelte:head>
  <title>Sparkly</title>
</svelte:head>

<main>
  <h1>Sparkly</h1>
</main>
