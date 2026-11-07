import React, { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { I18nProvider } from '../src/contexts/i18n-context'
import { MediaField } from '../src/components/ui/media-field'
import type { MediaPickerValue } from '../src/components/ui/media-picker'
import '../src/index.css'

const value: MediaPickerValue = { id: 1, url: 'uploads/example.png', path: 'uploads/example.png', original_name: 'example.png', size: 10, type: 'image' }

const Fixture: React.FC = () => {
  const [media, setMedia] = useState<MediaPickerValue | null>(value)
  return <I18nProvider><main><MediaField value={media} onChange={setMedia} type="image" label="Cover" /></main></I18nProvider>
}

createRoot(document.getElementById('root')!).render(<Fixture />)
