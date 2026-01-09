<script lang="ts">
  import * as Tooltip from "$lib/components/ui/tooltip/index.js";
  let { children, context } = $props();
</script>

<Tooltip.Root>
  <Tooltip.Trigger>
    {@render children?.()}
  </Tooltip.Trigger>
  <Tooltip.Content>
    <p>{context}</p>
  </Tooltip.Content>
</Tooltip.Root>
