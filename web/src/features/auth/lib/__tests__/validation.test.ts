/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import {
  cleanBackupCode,
  formatBackupCode,
  isValidBackupCode,
} from '../validation'

describe('backup code input normalization', () => {
  test.each([
    ['', ''],
    ['abcd', 'ABCD'],
    ['ab cd-12!34', 'ABCD-1234'],
    ['abcd12345678', 'ABCD-1234'],
    ['é中a😀b\nc\td1234', 'ABCD-1234'],
  ])('formats %j as %j', (input, expected) => {
    expect(formatBackupCode(input)).toBe(expected)
  })

  test.each([
    ['', ''],
    ['ABCD-1234', 'ABCD1234'],
    ['--AB-CD--12-34--', 'ABCD1234'],
    ['ab cd—12_34!', 'ab cd—12_34!'],
  ])('removes only ASCII hyphens from %j', (input, expected) => {
    expect(cleanBackupCode(input)).toBe(expected)
  })

  test('formatting a partial code does not make it valid', () => {
    expect(isValidBackupCode(formatBackupCode('abc'))).toBe(false)
    expect(isValidBackupCode(formatBackupCode('abcd1234'))).toBe(true)
  })
})
