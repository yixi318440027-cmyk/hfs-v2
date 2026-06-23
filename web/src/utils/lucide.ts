// Lucide icons utility
// Call lucide.createIcons() after DOM updates to re-render data-lucide elements.
// In templates, use: <i data-lucide="folder" class="lucide-icon"></i>

declare global {
  interface Window {
    lucide: {
      createIcons: (options?: { attrs?: Record<string, string> }) => void
    }
  }
}

let initDone = false

export function initLucide() {
  if (initDone) return
  if (window.lucide) {
    window.lucide.createIcons({
      attrs: {
        'stroke-width': '1.5',
        'width': '16',
        'height': '16',
      },
    })
    initDone = true
  }
}

export function refreshLucide() {
  if (window.lucide) {
    window.lucide.createIcons({
      attrs: {
        'stroke-width': '1.5',
        'width': '16',
        'height': '16',
      },
    })
  }
}
