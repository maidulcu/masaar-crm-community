export function triggerSync() {
  if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
    navigator.serviceWorker.controller.postMessage({ type: 'SYNC_NOW' })
  }
}

export async function getPendingCount(): Promise<number> {
  if (!('serviceWorker' in navigator) || !navigator.serviceWorker.controller) return 0
  return new Promise((resolve) => {
    const handler = (event: MessageEvent) => {
      if (event.data?.type === 'QUEUE_COUNT') {
        navigator.serviceWorker.removeEventListener('message', handler)
        resolve(event.data.count)
      }
    }
    const controller = navigator.serviceWorker.controller
    if (!controller) { resolve(0); return }
    navigator.serviceWorker.addEventListener('message', handler)
    controller.postMessage({ type: 'GET_QUEUE_COUNT' })
    setTimeout(() => {
      navigator.serviceWorker.removeEventListener('message', handler)
      resolve(0)
    }, 1000)
  })
}
