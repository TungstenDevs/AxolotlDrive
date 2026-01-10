<script lang="ts">
  import "./layout.css";
  import favicon from "$lib/assets/favicon.svg";
  import { ModeWatcher } from "mode-watcher";
  import { SidebarProvider, SidebarTrigger } from "$lib/components/ui/sidebar";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import TooltipProvider from "$lib/components/ui/tooltip/tooltip-provider.svelte";

  let { children } = $props();
</script>

<ModeWatcher defaultMode="system" />
<title>AxolotlDrive - Self Hosted File Sharing</title>
<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<TooltipProvider delayDuration={0}>
  <SidebarProvider>
    <AppSidebar />
    <main class="w-full">
      {@render children?.()}
    </main>
  </SidebarProvider>
</TooltipProvider>
