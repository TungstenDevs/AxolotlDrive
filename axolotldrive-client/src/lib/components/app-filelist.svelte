<script lang="ts" generics="TData">
  import type { FileAndFolder } from "$lib/types/types";
  import { Checkbox } from "$lib/components/ui/checkbox/index.js";
  import { ScrollArea } from "$lib/components/ui/scroll-area/index.js";
  import { FolderOpen, File } from "lucide-svelte";
  import { fileTableData } from "$lib/utils/filecollumn";

  type Props = {
    data?: FileAndFolder[];
  };

  let { data = fileTableData }: Props = $props();
  let selectedIds: Set<string> = $state(new Set());

  function toggleSelect(id: string) {
    if (selectedIds.has(id)) {
      selectedIds.delete(id);
    } else {
      selectedIds.add(id);
    }
    selectedIds = selectedIds;
  }

  function toggleAll() {
    if (selectedIds.size === data.length) {
      selectedIds.clear();
    } else {
      data.forEach((item) => selectedIds.add(item.id));
    }
    selectedIds = selectedIds;
  }

  function formatDate(date: Date): string {
    return new Date(date).toLocaleDateString();
  }

  function formatSize(size: number): string {
    if (size === 0) return "-";
    return `${size} MB`;
  }
</script>

<div class="w-full">
  <div
    class="border-b bg-muted/50 p-4 flex items-center gap-3 font-semibold text-sm sticky top-0"
  >
    <Checkbox
      checked={selectedIds.size === data.length && data.length > 0}
      indeterminate={selectedIds.size > 0 && selectedIds.size < data.length}
      onchange={toggleAll}
      aria-label="Select all items"
    />
    <div class="flex-1">Name</div>
    <div class="w-20">Type</div>
    <div class="w-24">Size</div>
    <div class="w-32">Created</div>
    <div class="w-32 hidden md:inline">Updated</div>
  </div>

  <ScrollArea class="h-[calc(100vh-200px)]">
    {#if data.length === 0}
      <div class="p-8 text-center text-muted-foreground">
        No files or folders found.
      </div>
    {:else}
      <div class="divide-y">
        {#each data as item (item.id)}
          <div
            class="p-4 flex items-center gap-3 hover:bg-muted/50 transition-colors cursor-pointer group"
            class:bg-muted={selectedIds.has(item.id)}
          >
            <Checkbox
              checked={selectedIds.has(item.id)}
              onchange={() => toggleSelect(item.id)}
              aria-label="Select item"
            />

            <div class="flex-1 flex items-center gap-2 min-w-0">
              {#if item.type === "folder"}
                <FolderOpen size={18} class="flex-shrink-0 text-yellow-600" />
                <a
                  href={`/folder/${item.id}`}
                  class="text-blue-600 hover:underline truncate"
                >
                  {item.name}
                </a>
              {:else}
                <File size={18} class="flex-shrink-0 text-gray-500" />
                <span class="truncate">{item.name}</span>
              {/if}
            </div>

            <div class="w-20 text-sm capitalize">{item.type}</div>
            <div class="w-24 text-sm">{formatSize(item.size)}</div>
            <div class="w-32 text-sm text-muted-foreground">
              {formatDate(item.createdAt)}
            </div>
            <div class="w-32 hidden md:inline text-sm text-muted-foreground">
              {formatDate(item.updatedAt)}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </ScrollArea>
</div>
