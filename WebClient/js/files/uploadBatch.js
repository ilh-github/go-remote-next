/**
 * files/uploadBatch.js - reporting a multi-file upload where some files landed.
 *
 * The panel uploads the operator's whole selection in one go. A bare loop that
 * awaits each file reports only the first error it hits, which hides two things
 * the operator needs: that the earlier files are already on the host, and which
 * one failed. They are left guessing whether to retry the batch, and a retry
 * re-uploads files that are already there.
 *
 * The rule is: run the whole batch, remember the first failure, and report both
 * halves. A batch where nothing landed is a plain failure and is rethrown so the
 * existing error path handles it.
 */

/**
 * @param {Array<{name: string}>} files the selection, in the order picked
 * @param {(file: object) => Promise<void>} uploadOne uploads a single file
 * @returns {Promise<{succeeded: string[], failure: Error|null}>}
 */
export async function runUploadBatch(files, uploadOne) {
  const succeeded = []
  let failure = null
  for (const file of files) {
    try {
      await uploadOne(file)
      succeeded.push(file.name)
    } catch (error) {
      // The first failure is the one worth naming: the rest of the batch is
      // still attempted, so reporting the last error would point at whichever
      // file happened to be last rather than at what broke first.
      if (!failure) failure = error instanceof Error ? error : new Error(String(error))
    }
  }
  return { succeeded, failure }
}

/**
 * The status key and parameters for a finished batch.
 *
 * @param {{succeeded: string[], failure: Error|null}} result
 * @param {number} total how many files were selected
 * @returns {{key: string, params: object, isError: boolean}|null} null when the
 *   batch was empty and there is nothing to report
 */
export function uploadBatchStatus(result, total) {
  const { succeeded, failure } = result
  if (total === 0) return null
  if (!failure) {
    return { key: 'files.uploadedCount', params: { count: succeeded.length }, isError: false }
  }
  if (succeeded.length === 0) {
    // Nothing landed, so this is a plain failure and the caller rethrows it
    // rather than rendering a partial-success message over a total loss.
    return null
  }
  return {
    key: 'files.uploadPartial',
    params: {
      done: succeeded.length,
      total,
      name: failure.message || String(failure),
    },
    isError: true,
  }
}
