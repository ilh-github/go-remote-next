// QD-909: reading the data-channel audit trail from the client.
//
// A session's detail view already shows what it produced - its recordings and
// its access log. The transfers that happened over the Remoting data channel are
// the third kind of record, and until now there was no way to read them back
// from the client at all: only the admin console could.
//
// What this deliberately does NOT claim is that the rows belong to one session.
// The audit row's resource id is the transfer id, not a session id, and the
// server has no session filter - so "the transfers in this session" is not a
// question the current contract can answer. Loading the account's own trail and
// labelling it as the account's is honest; loading it and labelling it as the
// session's would be a fabrication that reads as evidence.
//
// The composable exists so the loading rules can be tested without mounting a
// component. The rules that matter are about honesty rather than about fetching:
// a read that fails must say so rather than show an empty list, because "no
// transfers" and "we could not ask" are different facts and only one of them is
// evidence.

import { ref } from 'vue'
import { describeAuditRows } from '../api/auditLogModel.js'

/**
 * @param {{fetchAuditLogs: Function}} [api] the client to read through. Omit it
 *   and the real one is resolved on first use.
 * @returns {{
 *   rows: import('vue').Ref<Array<object>>,
 *   loading: import('vue').Ref<boolean>,
 *   error: import('vue').Ref<string>,
 *   loaded: import('vue').Ref<boolean>,
 *   load: () => Promise<void>,
 *   reset: () => void,
 * }}
 */
export function useSessionAudit(api) {
  const rows = ref([])
  const loading = ref(false)
  const error = ref('')
  // Where the current rows came from. `live` is the table, `archive` is the
  // per-day files retention moves expired rows into. An owner looking at an empty
  // panel needs to know which one they are looking at, because "nothing was
  // transferred" and "nothing is in the live table any more" are different facts.
  const source = ref('live')
  // Whether the current rows came from a completed read. An empty list with
  // `loaded` true means "this account moved nothing"; an empty list with
  // `loaded` false means "we never asked", and the two must not render alike.
  const loaded = ref(false)

  function reset() {
    rows.value = []
    loading.value = false
    error.value = ''
    source.value = 'live'
    loaded.value = false
  }

  /**
   * Loads the caller's own data-channel audit rows.
   *
   * The account's trail, not one session's - see the note at the top of the file
   * for why that distinction is kept rather than papered over.
   */
  async function load() {
    loading.value = true
    error.value = ''
    try {
      const client = await resolveClient(api)
      rows.value = describeAuditRows(await readEveryPage(client))
      loaded.value = true
      source.value = 'live'
      // An empty live table is not the whole answer once retention has run: the
      // rows may have been archived rather than never having existed. Ask the
      // archive only in that case, so a populated panel costs one request.
      if (rows.value.length === 0) {
        await loadFromArchive()
      }
    } catch (err) {
      // The failure is kept and the rows are cleared: leaving the previous
      // read's transfers on screen would be a worse lie than an error message.
      rows.value = []
      loaded.value = false
      error.value = err?.message || String(err)
      source.value = 'live'
    } finally {
      loading.value = false
    }
  }

// The server pages at 200 and hands back a `next` cursor. Asking for one page
// and stopping there would show an account with more transfers than that only
// its newest slice, with nothing on screen saying there is more - the owner
// reading "these are my transfers" would actually be reading "these are my
// newest transfers", and the difference matters exactly when they are looking
// for something old.
//
// The page size is the server's maximum rather than a number of our own, so a
// busy account costs as few requests as the contract allows. The loop is bounded
// by the cursor running out, and a page that fails after earlier pages succeeded
// keeps those rows: a partial read is more honest than an empty one, and the
// error path below is what handles a read that fails from the start.
async function readEveryPage(client, limit = 200) {
  const collected = []
  let cursor = ''
  for (;;) {
    const response = await client.fetchAuditLogs({ limit, cursor })
    if (!response || response.ok !== true) {
      if (collected.length === 0) {
        throw new Error(response?.error || 'the audit read failed')
      }
      return collected
    }
    const items = Array.isArray(response.data?.items) ? response.data.items : []
    collected.push(...items)
    const next = typeof response.data?.next === 'string' ? response.data.next : ''
    if (!next || next === cursor) return collected
    cursor = next
  }
}

  /**
   * Reads the archived half of the trail.
   *
   * A failure here does not fail the load: the live read already succeeded, and
   * an unset archive directory is a configuration fact rather than something the
   * owner did wrong. The rows simply stay empty and stay labelled `live`.
   */
  async function loadFromArchive() {
    try {
      const client = await resolveClient(api)
      const response = await client.fetchArchivedAuditLogs({ limit: 100 })
      if (!response || response.ok !== true) return
      const archived = describeAuditRows(response.data?.items ?? [])
      if (archived.length > 0) {
        rows.value = archived
        source.value = 'archive'
      }
    } catch {
      // Deliberately swallowed: see above.
    }
  }

  return { rows, loading, error, source, loaded, load, reset }
}

// Cached across composable instances so the real module is imported at most once.
let cachedClient = null

// The real client, resolved on first use rather than at import time.
//
// Importing `api/userApi` at module scope would pull its side effects into every
// consumer - it reads localStorage in its constructor, so a Node test with no DOM
// would throw before any test ran. Resolving it lazily means a caller that passes
// its own client never touches the real module at all, which is what makes this
// composable testable without a browser.
async function resolveClient(api) {
  if (api) return api
  if (!cachedClient) {
    const module = await import('../api/userApi.js')
    cachedClient = module.userApi
  }
  return cachedClient
}
