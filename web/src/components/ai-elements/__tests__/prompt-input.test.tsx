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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'

import {
  PromptInput,
  PromptInputAttachments,
  PromptInputTextarea,
} from '../prompt-input'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

it('keeps the attachment for retry when blob conversion fails before submission', async () => {
  vi.stubGlobal(
    'URL',
    Object.assign(class extends URL {}, {
      createObjectURL: vi.fn(() => 'blob:attachment'),
      revokeObjectURL: vi.fn(),
    })
  )
  const fetchBlob = vi.fn().mockRejectedValue(new Error('Conversion failed'))
  vi.stubGlobal('fetch', fetchBlob)
  const onSubmit = vi.fn()

  render(
    <PromptInput onSubmit={onSubmit}>
      <PromptInputAttachments>
        {(attachment) => <span>{attachment.filename}</span>}
      </PromptInputAttachments>
      <PromptInputTextarea aria-label='Message' />
      <button type='submit'>Send</button>
    </PromptInput>
  )

  fireEvent.change(screen.getByLabelText('Upload files'), {
    target: { files: [new File(['data'], 'example.txt')] },
  })
  fireEvent.change(screen.getByRole('textbox', { name: 'Message' }), {
    target: { value: 'Please use the attachment' },
  })
  expect(screen.getByText('example.txt')).toBeVisible()

  fireEvent.click(screen.getByRole('button', { name: 'Send' }))
  await waitFor(() =>
    expect(screen.getByRole('textbox', { name: 'Message' })).toHaveValue(
      'Please use the attachment'
    )
  )
  expect(fetchBlob).toHaveBeenCalledWith('blob:attachment')
  expect(screen.getByText('example.txt')).toBeVisible()
  expect(onSubmit).not.toHaveBeenCalled()
})

it('preserves a new draft typed while blob conversion is pending and then fails', async () => {
  vi.stubGlobal(
    'URL',
    Object.assign(class extends URL {}, {
      createObjectURL: vi.fn(() => 'blob:attachment'),
      revokeObjectURL: vi.fn(),
    })
  )
  let rejectFetch: (error: Error) => void = () => undefined
  const pendingFetch = new Promise<Response>((_, reject) => {
    rejectFetch = reject
  })
  const fetchBlob = vi.fn().mockReturnValue(pendingFetch)
  vi.stubGlobal('fetch', fetchBlob)
  const onSubmit = vi.fn()

  render(
    <PromptInput onSubmit={onSubmit}>
      <PromptInputAttachments>
        {(attachment) => <span>{attachment.filename}</span>}
      </PromptInputAttachments>
      <PromptInputTextarea aria-label='Message' />
      <button type='submit'>Send</button>
    </PromptInput>
  )

  fireEvent.change(screen.getByLabelText('Upload files'), {
    target: { files: [new File(['data'], 'example.txt')] },
  })
  const message = screen.getByRole('textbox', { name: 'Message' })
  fireEvent.change(message, { target: { value: 'Original draft' } })
  fireEvent.click(screen.getByRole('button', { name: 'Send' }))
  expect(fetchBlob).toHaveBeenCalledWith('blob:attachment')
  expect(message).toHaveValue('')

  fireEvent.change(message, { target: { value: 'New draft' } })
  await act(async () => {
    rejectFetch(new Error('Conversion failed'))
    await pendingFetch.catch(() => undefined)
  })

  expect(message).toHaveValue('New draft')
  expect(screen.getByText('example.txt')).toBeVisible()
  expect(onSubmit).not.toHaveBeenCalled()
})
