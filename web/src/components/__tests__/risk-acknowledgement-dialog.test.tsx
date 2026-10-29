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
import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { RiskAcknowledgementDialog } from '../risk-acknowledgement-dialog'

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

test('duplicate confirmation segments keep independent inputs and require both matches', async () => {
  const onConfirm = vi.fn()
  render(
    <RiskAcknowledgementDialog
      open
      onOpenChange={() => undefined}
      title='Confirm the operation'
      description='Both confirmations are required.'
      requiredTextParts={[
        { type: 'static', text: 'Confirm: ' },
        { type: 'input', text: 'ACCEPT', placeholder: 'First confirmation' },
        { type: 'static', text: 'Confirm: ' },
        { type: 'input', text: 'ACCEPT', placeholder: 'Second confirmation' },
      ]}
      onConfirm={onConfirm}
    />
  )
  await act(() => vi.advanceTimersByTimeAsync(0))

  const confirm = screen.getByRole('button', { name: 'Confirm' })
  expect(confirm).toBeDisabled()
  fireEvent.change(screen.getByPlaceholderText('First confirmation'), {
    target: { value: 'ACCEPT' },
  })
  expect(confirm).toBeDisabled()
  fireEvent.change(screen.getByPlaceholderText('Second confirmation'), {
    target: { value: 'ACCEPT' },
  })
  expect(confirm).toBeEnabled()
  fireEvent.click(confirm)
  expect(onConfirm).toHaveBeenCalledOnce()
})

test('updating static confirmation text preserves typed input and focus in its logical slot', async () => {
  const props = {
    open: true,
    onOpenChange: () => undefined,
    title: 'Confirm the operation',
    description: 'Review the confirmation.',
    onConfirm: vi.fn(),
  }
  const { rerender } = render(
    <RiskAcknowledgementDialog
      {...props}
      requiredTextParts={[
        { type: 'static', text: 'Confirm: ' },
        { type: 'input', text: 'ACCEPT', placeholder: 'Confirmation' },
      ]}
    />
  )
  await act(() => vi.advanceTimersByTimeAsync(0))

  const input = screen.getByPlaceholderText('Confirmation')
  fireEvent.change(input, { target: { value: 'ACCEPT' } })
  input.focus()
  rerender(
    <RiskAcknowledgementDialog
      {...props}
      requiredTextParts={[
        { type: 'static', text: 'Please confirm: ' },
        { type: 'input', text: 'ACCEPT', placeholder: 'Confirmation' },
      ]}
    />
  )

  expect(screen.getByPlaceholderText('Confirmation')).toHaveValue('ACCEPT')
  expect(screen.getByPlaceholderText('Confirmation')).toHaveFocus()
  expect(screen.getByRole('button', { name: 'Confirm' })).toBeEnabled()
})
