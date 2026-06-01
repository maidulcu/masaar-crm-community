'use client'
import { useEffect, useRef } from 'react'
import clsx from 'clsx'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  children: React.ReactNode
}

export function Modal({ open, onClose, title, children }: ModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return

    if (open) {
      dialog.showModal()
    } else {
      dialog.close()
    }
  }, [open])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onClose()
      }
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [open, onClose])

  return (
    <dialog
      ref={dialogRef}
      className={clsx(
        'backdrop:bg-surface-900/50 backdrop:backdrop-blur-sm bg-transparent p-0 m-auto',
        'open:animate-fade-in'
      )}
      onClick={(e) => {
        if (e.target === dialogRef.current) onClose()
      }}
    >
      <div className="bg-white rounded-2xl shadow-pop ring-1 ring-surface-200/60 w-full max-w-lg p-6 animate-scale-in">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-surface-900 tracking-tight">{title}</h2>
          <button
            onClick={onClose}
            aria-label="Close"
            className="inline-flex items-center justify-center w-8 h-8 rounded-lg text-surface-400 hover:text-surface-900 hover:bg-surface-100 transition-colors"
          >
            <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        </div>
        {children}
      </div>
    </dialog>
  )
}

interface FormFieldProps {
  label: string
  children: React.ReactNode
}

export function FormField({ label, children }: FormFieldProps) {
  return (
    <div>
      <label className="block text-[13px] font-medium text-surface-700 mb-1.5">{label}</label>
      {children}
    </div>
  )
}

export function FormError({ error }: { error: string }) {
  return (
    <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg mt-2">{error}</p>
  )
}
