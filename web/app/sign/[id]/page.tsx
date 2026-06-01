'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api } from '@/lib/api'

interface Signature {
  id: string
  document_id: string
  signer_name: string
  signer_email: string
  signature_status: string
  signed_at?: string
}

interface Document {
  id: string
  document_title: string
  document_type: string
  file_url?: string
}

interface SignatureData {
  signature: Signature
  document: Document
}

type PageState = 'loading' | 'ready' | 'signing' | 'already_signed' | 'signed' | 'error'

export default function PublicSignPage() {
  const { id } = useParams<{ id: string }>()

  const [data, setData] = useState<SignatureData | null>(null)
  const [pageState, setPageState] = useState<PageState>('loading')
  const [errorMsg, setErrorMsg] = useState('')
  const [agreed, setAgreed] = useState(false)

  useEffect(() => {
    if (!id) return
    api.public.getSignature(id)
      .then((d: SignatureData) => {
        setData(d)
        if (d.signature.signature_status === 'signed') {
          setPageState('already_signed')
        } else {
          setPageState('ready')
        }
      })
      .catch((err) => {
        setErrorMsg(err.message || 'Failed to load signature request')
        setPageState('error')
      })
  }, [id])

  async function handleSign() {
    if (!agreed || !id) return
    setPageState('signing')
    try {
      await api.public.sign(id)
      setPageState('signed')
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : 'Signing failed')
      setPageState('error')
    }
  }

  if (pageState === 'loading') {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-4"></div>
          <p className="text-gray-600">Loading signature request...</p>
        </div>
      </div>
    )
  }

  if (pageState === 'error') {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
        <div className="max-w-md w-full bg-white rounded-2xl shadow-lg p-8 text-center">
          <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Something went wrong</h2>
          <p className="text-gray-600">{errorMsg || 'This signature link is invalid or has expired.'}</p>
        </div>
      </div>
    )
  }

  if (pageState === 'already_signed' || pageState === 'signed') {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
        <div className="max-w-md w-full bg-white rounded-2xl shadow-lg p-8 text-center">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            {pageState === 'signed' ? 'Signed Successfully!' : 'Already Signed'}
          </h2>
          <p className="text-gray-600 mb-6">
            {pageState === 'signed'
              ? 'Your signature has been recorded. Thank you!'
              : 'This document has already been signed.'}
          </p>
          {data?.document.document_title && (
            <div className="bg-gray-50 rounded-xl p-4 mb-6">
              <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">Document</p>
              <p className="font-semibold text-gray-900">{data.document.document_title}</p>
            </div>
          )}
          {data?.signature.signed_at && (
            <p className="text-sm text-gray-500">
              Signed on {new Date(data.signature.signed_at).toLocaleString()}
            </p>
          )}
        </div>
      </div>
    )
  }

  // ready or signing state
  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
      <div className="max-w-lg w-full">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="w-12 h-12 bg-blue-100 rounded-xl flex items-center justify-center mx-auto mb-3">
            <svg className="w-6 h-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-gray-900">Document Signature Request</h1>
          <p className="text-gray-600 mt-1">You have been asked to sign the following document</p>
        </div>

        {/* Document Card */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-6 mb-6">
          <div className="mb-4">
            <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">Document</p>
            <h2 className="text-xl font-semibold text-gray-900">{data?.document.document_title}</h2>
            <span className="inline-block mt-1 text-xs px-2 py-0.5 bg-gray-100 text-gray-600 rounded capitalize">
              {data?.document.document_type}
            </span>
          </div>

          <div className="border-t border-gray-100 pt-4">
            <p className="text-xs text-gray-500 uppercase tracking-wide mb-1">Requested Signer</p>
            <p className="font-medium text-gray-900">{data?.signature.signer_name}</p>
            <p className="text-sm text-gray-500">{data?.signature.signer_email}</p>
          </div>

          {data?.document.file_url && (
            <div className="border-t border-gray-100 pt-4 mt-4">
              <a
                href={data.document.file_url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-sm text-blue-600 hover:underline font-medium"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
                View Document
              </a>
            </div>
          )}
        </div>

        {/* Agreement and Sign */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-6">
          <label className="flex items-start gap-3 cursor-pointer mb-6">
            <input
              type="checkbox"
              checked={agreed}
              onChange={(e) => setAgreed(e.target.checked)}
              className="mt-0.5 w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">
              I, <strong>{data?.signature.signer_name}</strong>, confirm that I have read and agree to sign this document. I understand that my electronic signature is legally binding.
            </span>
          </label>

          <button
            onClick={handleSign}
            disabled={!agreed || pageState === 'signing'}
            className="w-full py-3 bg-blue-600 text-white rounded-xl font-semibold text-base hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {pageState === 'signing' ? (
              <span className="flex items-center justify-center gap-2">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                Signing...
              </span>
            ) : (
              'Sign Document'
            )}
          </button>

          <p className="text-xs text-gray-400 text-center mt-4">
            Your IP address and browser information will be recorded for verification purposes.
          </p>
        </div>
      </div>
    </div>
  )
}
