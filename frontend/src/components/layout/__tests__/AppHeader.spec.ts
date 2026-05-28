import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppHeader user dropdown', () => {
  it('does not duplicate chat launch entries that live in the left user menu', () => {
    expect(componentSource).not.toContain("import { launchLobeHub, launchOnyx } from '@/api/onyx'")
    expect(componentSource).not.toContain('showLobeHubMenu')
    expect(componentSource).not.toContain('showOnyxMenu')
    expect(componentSource).not.toContain('handleLobeHubLaunch')
    expect(componentSource).not.toContain('handleOnyxLaunch')
    expect(componentSource).not.toContain('launchLobeHub()')
    expect(componentSource).not.toContain('launchOnyx()')

    const dropdownBlock = componentSource.match(/<div class="py-1">[\s\S]*?<\/div>\n\n              <!-- Contact Support/)?.[0] ?? ''
    expect(dropdownBlock).not.toContain("t('nav.lobehub')")
    expect(dropdownBlock).not.toContain("t('nav.chat')")
  })
})
