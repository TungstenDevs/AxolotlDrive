<script lang="ts">
  import AppFiledatatable from "$lib/components/app-filedatatable.svelte";
  import AppSearch from "$lib/components/app-search.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import AppTooltip from "$lib/components/utils/app-tooltip.svelte";
  import { fileTableColumns, fileTableData } from "$lib/utils/filecollumn";
  import {
    FolderPlus,
    InfoIcon,
    MenuIcon,
    ShareIcon,
    UploadCloud,
  } from "lucide-svelte";
  import Grid_3x3 from "lucide-svelte/icons/grid-3x3";

  let layout: "grid" | "list" = $state("list");
  let allSelected: boolean = $state(false);
</script>

<div class="w-full min-h-screen flex flex-col">
  <div class="flex items-center justify-between p-3 border-b w-full">
    <AppSearch />
    <div class="flex gap-2">
      <Button variant="link" href="/auth">Sign In</Button>
      <Button variant="default" href="/auth">Sign Up</Button>
    </div>
  </div>

  <div class="flex items-center gap-2 w-full border-b p-3 justify-between">
    <h1 class="text-md font-bold flex items-center gap-2">
      Public Files <AppTooltip
        context="These are files shared publicly by users. You can do anything with them"
      >
        <InfoIcon size={16} />
      </AppTooltip>
    </h1>
    <div class="flex items-center gap-3">
      <h1 class="flex items-center text-sm gap-2">
        <AppTooltip context="Create new folder in public area.">
          <FolderPlus size={20} />
        </AppTooltip>
        <span>|</span>
      </h1>
      <h1 class="flex items-center text-sm gap-2">
        <AppTooltip context="Upload a new file to public area.">
          <UploadCloud size={20} />
        </AppTooltip>
        <span>|</span>
      </h1>
      <h1 class="flex items-center text-sm gap-2">
        <AppTooltip context="Share this file with others.">
          <ShareIcon size={20} />
        </AppTooltip>
        <span>|</span>
      </h1>
      <h1 class="flex items-center text-sm gap-2">
        <AppTooltip context="Toggle layout view.">
          {#if layout == "list"}
            <MenuIcon size={20} />
          {:else}
            <Grid_3x3 size={20} />
          {/if}
        </AppTooltip>
      </h1>
    </div>
  </div>

  <!-- <div class="w-full p-3 flex items-center justify-between border-b">
    <div class="flex gap-2 items-center">
      <Checkbox
        id="files"
        checked={allSelected}
        onchange={() => (allSelected = !allSelected)}
      />
      <label for="files" class="text-sm font-semibold cursor-pointer"
        >Select All</label
      >
    </div>
  </div> -->
  <AppFiledatatable data={fileTableData} columns={fileTableColumns} />
</div>
