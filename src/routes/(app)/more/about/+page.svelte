<script lang="ts">
  import Logo from "$lib/components/Logo.svelte";
  const links = [
    { name: "Issue", href: "https://github.com/nextxm/paisa/issues", icon: "fas fa-bug" },
    {
      name: "Discussions",
      href: "https://github.com/nextxm/paisa/discussions",
      icon: "fa-regular fa-comments"
    },
    {
      name: "Source Code",
      href: "https://github.com/nextxm/paisa",
      icon: "fa-solid fa-code"
    },
    {
      name: "Upstream Project",
      href: "https://github.com/ananthakumaran/paisa",
      icon: "fa-solid fa-code-branch"
    },
    {
      name: "License (AGPL-3.0-or-later)",
      href: "https://www.gnu.org/licenses/agpl-3.0.html",
      icon: "fa-solid fa-scale-balanced"
    },
    { name: "Documentation", href: "https://nextxm.github.io/paisa/", icon: "fa-solid fa-book" },
    {
      name: "Releases",
      href: "https://github.com/nextxm/paisa/releases",
      icon: "fa-solid fa-download"
    }
  ];

  function externalLink(url: string) {
    if (window.runtime) {
      window.runtime.BrowserOpenURL(url);
    } else {
      window.open(url, "_blank");
    }
  }

  const buildInfo = __BUILD_INFO__;
</script>

<section class="section">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <div class="box has-text-centered px-3 mx-auto" style="max-width: 400px;">
          <div><Logo size={128} /></div>
          <div class="is-size-3 is-primary-color">Paisa</div>
          <div>
            Version: <b>{buildInfo.version}</b>
          </div>
          <div class="is-size-7 mt-2 has-text-grey">
            Build: {buildInfo.buildDate.substring(0, 10)}<br />
            Branch: {buildInfo.branch} ({buildInfo.commitHash})
            {#if buildInfo.tag}
              <br />Tag: {buildInfo.tag}
            {/if}
          </div>
          <div class="is-size-7 mt-2">
            Forked from ananthakumaran/paisa, maintained at nextxm/paisa under GNU
            AGPL-3.0-or-later.
          </div>
        </div>

        <div class="box px-3 mx-auto" style="max-width: 400px;">
          <h3 class="is-size-5 mb-1">Links</h3>
          <ul>
            {#each links as link}
              <li>
                <a href={link.href} on:click|preventDefault={(_e) => externalLink(link.href)}>
                  <span class="icon is-small">
                    <i class={link.icon} />
                  </span>
                  <span>{link.name}</span>
                </a>
              </li>
            {/each}
          </ul>
        </div>
      </div>
    </div>
  </div>
</section>
