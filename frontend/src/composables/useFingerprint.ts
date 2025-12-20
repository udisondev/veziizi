import FingerprintJS from '@fingerprintjs/fingerprintjs'

let cachedFingerprint: string | null = null
let fpPromise: Promise<string> | null = null

/**
 * Get device fingerprint for fraud detection.
 * Uses FingerprintJS library to generate a unique device identifier.
 * The fingerprint is cached after first load for the session.
 */
export async function getFingerprint(): Promise<string> {
  // Return cached fingerprint if available
  if (cachedFingerprint) {
    return cachedFingerprint
  }

  // Prevent multiple simultaneous loads
  if (fpPromise) {
    return fpPromise
  }

  fpPromise = (async () => {
    try {
      const fp = await FingerprintJS.load()
      const result = await fp.get()
      cachedFingerprint = result.visitorId
      return cachedFingerprint
    } catch (error) {
      console.error('Failed to get fingerprint:', error)
      return ''
    } finally {
      fpPromise = null
    }
  })()

  return fpPromise
}

/**
 * Preload fingerprint in the background.
 * Call this early (e.g., on app mount) to reduce latency during login/registration.
 */
export function preloadFingerprint(): void {
  getFingerprint().catch(() => {})
}
