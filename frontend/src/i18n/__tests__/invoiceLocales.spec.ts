import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('invoice locale copy', () => {
  it('escapes literal @ signs so vue-i18n does not parse them as linked messages', () => {
    expect(zh.payment.invoice.emailPlaceholder).toBe("name{'@'}example.com")
    expect(en.payment.invoice.emailPlaceholder).toBe("name{'@'}example.com")
  })
})
