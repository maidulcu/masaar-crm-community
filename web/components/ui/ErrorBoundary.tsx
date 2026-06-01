'use client'
import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  message: string
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, message: '' }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, message: error.message }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-surface-50 p-8 text-center">
          <div className="max-w-md">
            <div className="mx-auto w-12 h-12 rounded-2xl bg-red-50 text-red-500 flex items-center justify-center mb-4">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M12 9v2m0 4h.01M5 19h14a2 2 0 001.84-2.75L13.74 4a2 2 0 00-3.48 0L3.16 16.25A2 2 0 005 19z" />
              </svg>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-surface-900 mb-2">
              حدث خطأ غير متوقع / Unexpected Error
            </h1>
            <p className="text-surface-500 text-sm mb-6">{this.state.message}</p>
            <button
              onClick={() => window.location.reload()}
              className="px-4 py-2.5 bg-primary-600 text-white rounded-xl text-sm font-medium shadow-card hover:bg-primary-700 hover:shadow-card-hover transition-all"
            >
              إعادة تحميل / Reload
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
