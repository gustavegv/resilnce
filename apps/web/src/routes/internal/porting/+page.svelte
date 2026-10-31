<script lang="ts">
  import { onMount } from 'svelte';
  import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';

  type Count = { source: number; mapped: number; pending: number; imported: number; retained: number; deletePlanned?: number; deleted?: number };
  type Issue = { table: string; key: unknown; code: string; severity: string; details: string };
  type Assignment = { sourceKey: unknown; group: string; name: string; position: number; weights: string[]; reps: number[]; restSeconds?: number };
  type Owner = { owner: string; ready: boolean; counts: Record<string, Count>; issues: Issue[]; assignments: Assignment[]; destinationObjectsCreated: number; error?: string };
  type Run = { id: string; actor: string; mode: string; ownerScope: string | null; status: string; createdAt: string; error: string | null; owners?: Owner[]; options: { sourceTimezone: string; deleteUnusedExercises?: boolean; deleteUnusedHistory?: boolean } };
  type Inventory = { owner: string; table: string; source: number; mapped: number };
  type SchemaStatus = { ready: boolean; canApply: boolean; migrations: { version: string; applied: boolean }[]; issues: string[] };

  let schema: SchemaStatus | null = null;
  let inventory: Inventory[] = [];
  let runs: Run[] = [];
  let selected: Run | null = null;
  let owner = '';
  let sourceTimezone = '';
  let deleteUnusedExercises = false;
  let deleteUnusedHistory = false;
  $: if (!deleteUnusedExercises) deleteUnusedHistory = false;
  let applyEnabled = false;
  let authorized = false;
  let busy = false;
  let error = '';
  let needsLogin = false;
  let polling = false;
  let disposed = false;

  async function request(path: string, body?: unknown) {
    const response = await fetch(`${PUBLIC_BACKEND_BASE_URL}/internal/porting/${path}`, {
      method: body === undefined ? 'GET' : 'POST',
      credentials: 'include',
      ...(body === undefined ? {} : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
    });
    if (!response.ok) {
      if (response.status === 401 || response.status === 403) {
        authorized = false;
        inventory = [];
        runs = [];
        selected = null;
        schema = null;
        needsLogin = response.status === 401;
      }
      throw new Error(await response.text());
    }
    return response.json();
  }

  async function overview() {
    const data = await request('overview');
    schema = data.schema;
    inventory = data.inventory;
    runs = data.runs;
    applyEnabled = data.applyEnabled;
    authorized = true;
    needsLogin = false;
  }

  async function inspect(id: string) {
    selected = await request(`runs/${id}`);
  }

  async function applyMigrations() {
    busy = true;
    error = '';
    try {
      schema = await request('schema/apply', {});
      await overview();
    } catch (e) { error = String(e); }
    finally { busy = false; }
  }

  async function start(mode: 'audit' | 'apply') {
    busy = true;
    error = '';
    try {
      const data = await request(mode === 'audit' ? 'audits' : 'runs', {
        owner: owner.trim(), options: { sourceTimezone: sourceTimezone.trim(), deleteUnusedExercises, deleteUnusedHistory: deleteUnusedExercises && deleteUnusedHistory },
      });
      await inspect(data.id);
      await overview();
    } catch (e) { error = String(e); }
    finally { busy = false; }
  }

  async function resume() {
    if (!selected) return;
    busy = true;
    error = '';
    try { await request(`runs/${selected.id}/resume`, {}); await inspect(selected.id); }
    catch (e) { error = String(e); }
    finally { busy = false; }
  }

  async function refresh() {
    if (polling || disposed) return;
    polling = true;
    try {
      await overview();
      if (selected) await inspect(selected.id);
      error = '';
    } catch (e) { error = String(e); }
    finally { polling = false; }
  }

  onMount(() => {
    void refresh();
    const timer = setInterval(() => {
      if (authorized && (selected?.status === 'queued' || selected?.status === 'running')) void refresh();
    }, 2000);
    return () => { disposed = true; clearInterval(timer); };
  });
</script>

<svelte:head><title>Legacy data porting · resilnce</title></svelte:head>

<main>
  <a href="/">← resilnce</a>
  <h1>Legacy data porting</h1>
  <p>Audit legacy accounts, inspect pending records, and follow resumable import jobs.</p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if needsLogin}
    <p>A fresh sign-in is required to establish your verified email.</p>
    <a href={`${PUBLIC_BACKEND_BASE_URL}/login/google`}>Sign in with Google</a>
  {/if}

  {#if authorized}
    <section aria-label="Database layout">
      <h2>Database layout</h2>
      {#if schema?.ready}
        <p role="status">The connected database has the required layout. Audits and imports are available.</p>
      {:else if schema?.canApply}
        <p>The connected database needs the additive migrations listed below.</p>
        <p>Apply them to create the exercise library and import bookkeeping tables. Existing workout data is preserved.</p>
        <button disabled={busy || polling} onclick={applyMigrations}>{busy ? 'Applying migrations…' : 'Apply additive migrations'}</button>
      {:else}
        <p>The database layout needs attention before audits or imports can run.</p>
      {/if}
      <ul>{#each schema?.migrations ?? [] as migration}
        <li><code>{migration.version}</code> · {migration.applied ? 'Applied' : 'Pending'}</li>
      {/each}</ul>
      {#if schema?.issues.length}
        <ul>{#each schema.issues as issue}<li class="error">{issue}</li>{/each}</ul>
      {/if}
      <button disabled={busy || polling} onclick={refresh}>Check database layout</button>
    </section>

    {#if schema?.ready}
    <section aria-label="Import controls">
      <h2>Run an audit or import</h2>
      <label>Account scope<input bind:value={owner} type="email" placeholder="Leave empty for all accounts" /></label>
      <label>Source timezone (optional)<input bind:value={sourceTimezone} placeholder="Europe/Stockholm" /></label>
      <p>Blank uses Stockholm time. Missing rest durations default to 0 seconds.</p>
      <label class="checkbox"><input type="checkbox" bind:checked={deleteUnusedExercises} /> Delete unused exercises</label>
      <label class="checkbox"><input type="checkbox" bind:checked={deleteUnusedHistory} disabled={!deleteUnusedExercises} /> Also delete their history</label>
      <p>Unused means no saved-session placement. History is preserved unless both options are checked. Audit shows the exercise and history deletion counts; import deletes the selected legacy records. Already imported data is preserved.</p>
      <p>Active legacy workouts and invalid source values keep affected accounts incomplete.</p>
      {#if !applyEnabled}<p>Imports are disabled until the deployment is configured for a rehearsal or reviewed cutover.</p>{/if}
      <div class="actions">
        <button disabled={busy} onclick={() => start('audit')}>Audit source data</button>
        <button disabled={busy || !applyEnabled} onclick={() => start('apply')}>Import pending records</button>
        <button disabled={busy || polling} onclick={refresh}>Refresh</button>
      </div>
      <p>Jobs run in the independent porting worker. Closing this page leaves a queued or running job available to resume.</p>
    </section>

    <section aria-label="Source inventory">
      <h2>Source inventory</h2>
      <div class="scroll"><table>
        <thead><tr><th>Account</th><th>Source table</th><th>Source rows</th><th>Mapped rows</th><th>Pending rows</th></tr></thead>
        <tbody>{#each inventory as item}
          <tr><td>{item.owner || 'Missing owner'}</td><td>{item.table}</td><td>{item.source}</td><td>{item.mapped}</td><td>{item.source - item.mapped}</td></tr>
        {/each}</tbody>
      </table></div>
    </section>

    <section aria-label="Recent runs">
      <h2>Recent runs</h2>
      {#if runs.length === 0}<p>No runs have been started.</p>{/if}
      {#each runs as run}
        <button class="run" onclick={async () => { try { await inspect(run.id); error = ''; } catch (e) { error = String(e); } }}>
          <strong>{run.mode} · {run.status}</strong>
          <span>{run.ownerScope || 'All accounts'} · {new Date(run.createdAt).toLocaleString()}</span>
          <small>{run.id} · {run.actor}</small>
        </button>
      {/each}
    </section>

    {#if selected}
      <section aria-label="Selected run">
        <h2>{selected.mode} · {selected.status}</h2>
        <p><code>{selected.id}</code></p>
        <p>Run settings: {selected.options.sourceTimezone || 'Europe/Stockholm'}; delete unused exercises {selected.options.deleteUnusedExercises ? 'enabled' : 'disabled'}; delete their history {selected.options.deleteUnusedHistory ? 'enabled' : 'disabled'}.</p>
        {#if selected.status === 'failed' || selected.status === 'incomplete'}<p>Retry uses these run settings. Start a new audit or import above to use different settings.</p>{/if}
        {#if selected.error}<p class="error">{selected.error}</p>{/if}
        {#if selected.status === 'queued'}<p>Waiting for the porting worker.</p>{/if}
        {#if selected.status === 'running'}<p>Worker is processing accounts. Progress refreshes automatically.</p>{/if}
        {#if selected.status === 'failed' || selected.status === 'incomplete'}
          <button disabled={busy || (selected.mode === 'apply' && !applyEnabled)} onclick={resume}>Retry unresolved work</button>
        {/if}
        {#each selected.owners ?? [] as account}
          <details open={!account.ready}>
            <summary>{account.owner || 'Missing owner'} · {account.ready ? 'Complete' : 'Incomplete'}</summary>
            {#if account.error}<p class="error">{account.error}</p>{/if}
            <p>Destination objects created in this attempt: {account.destinationObjectsCreated}</p>
            <div class="scroll"><table>
              <thead><tr><th>Source table</th><th>Source</th><th>Skipped mappings</th><th>Imported</th><th>Pending</th><th>Retained legacy</th><th>Planned deletions</th><th>Deleted</th></tr></thead>
              <tbody>{#each Object.entries(account.counts ?? {}) as [table, count]}
                <tr><td>{table}</td><td>{count.source}</td><td>{count.mapped}</td><td>{count.imported}</td><td>{count.pending}</td><td>{count.retained}</td><td>{count.deletePlanned ?? 0}</td><td>{count.deleted ?? 0}</td></tr>
              {/each}</tbody>
            </table></div>
            {#if account.assignments?.length}
              <h3>Variant assignments</h3>
              <ul>{#each account.assignments as assignment}
                <li><code>{JSON.stringify(assignment.sourceKey)}</code> → {assignment.group} / {assignment.name}; position {assignment.position}; targets {assignment.reps.join(', ')} reps at {assignment.weights.join(', ')} kg; rest {assignment.restSeconds ?? 0} seconds</li>
              {/each}</ul>
            {/if}
            <ul>{#each account.issues ?? [] as issue}
              <li class:error={issue.severity === 'error'}><strong>{issue.severity}: {issue.code}</strong> · {issue.table} <code>{JSON.stringify(issue.key)}</code> — {issue.details}</li>
            {/each}</ul>
          </details>
        {/each}
      </section>
    {/if}
    {/if}
  {/if}
</main>

<style>
  main { max-width: 1100px; margin: 0 auto; padding: 2rem 1rem 6rem; }
  h1 { font-size: 2rem; font-weight: 650; margin: 1rem 0; }
  h2 { font-size: 1.25rem; font-weight: 600; margin-bottom: 1rem; }
  h3 { font-weight: 600; margin-top: 1rem; }
  section { border: 1px solid #7775; border-radius: 0.75rem; padding: 1.25rem; margin-top: 1.5rem; }
  p { margin: 0.75rem 0; }
  label { display: block; margin: 1rem 0; }
  input { display: block; width: 100%; max-width: 36rem; border: 1px solid #8888; padding: 0.65rem; border-radius: 0.4rem; margin-top: 0.4rem; }
  .checkbox { display: flex; align-items: center; gap: 0.6rem; }
  .checkbox input { width: auto; margin: 0; }
  button { border: 1px solid #8888; padding: 0.6rem 1rem; border-radius: 0.4rem; cursor: pointer; text-align: left; }
  button:disabled { opacity: 0.5; cursor: default; }
  .actions { display: flex; flex-wrap: wrap; gap: 0.5rem; }
  .run { display: block; width: 100%; margin: 0.5rem 0; }
  .run span, .run small { display: block; overflow-wrap: anywhere; }
  .scroll { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
  th, td { text-align: left; border-bottom: 1px solid #8884; padding: 0.6rem; white-space: nowrap; }
  details { padding: 1rem 0; border-top: 1px solid #8884; }
  summary { cursor: pointer; font-weight: 600; }
  li { margin: 0.5rem 0; overflow-wrap: anywhere; }
  .error { color: #e15050; }
  code { overflow-wrap: anywhere; }
</style>
