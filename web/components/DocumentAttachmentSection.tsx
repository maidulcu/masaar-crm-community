'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'

interface Document {
  id: string
  document_title: string
  document_type: string
  file_url?: string
  signature_status: string
  created_at: string
}

interface Signature {
  id: string
  signer_name: string
  signer_email: string
  signature_status: string
  signed_at?: string
  created_at: string
}

interface DocumentAttachmentSectionProps {
  entityType: string
  entityId: string
  canEdit?: boolean
}

const DOC_TYPES = ['lease', 'contract', 'invoice', 'agreement', 'addendum', 'other']

export default function DocumentAttachmentSection({
  entityType,
  entityId,
  canEdit = false,
}: DocumentAttachmentSectionProps) {
  const [documents, setDocuments] = useState<Document[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedDoc, setExpandedDoc] = useState<string | null>(null)
  const [signatures, setSignatures] = useState<Record<string, Signature[]>>({})

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [creating, setCreating] = useState(false)
  const [createData, setCreateData] = useState({
    document_title: '',
    document_type: '',
    file_url: '',
  })

  const [sigFormDocId, setSigFormDocId] = useState<string | null>(null)
  const [signerData, setSignerData] = useState({ signerName: '', signerEmail: '' })
  const [sendingSignature, setSendingSignature] = useState(false)

  useEffect(() => {
    loadDocuments()
  }, [entityType, entityId])

  async function loadDocuments() {
    setLoading(true)
    try {
      const docs = await api.documents.list(entityType, entityId)
      setDocuments(Array.isArray(docs) ? (docs as Document[]) : [])
    } catch (err) {
      console.error('Failed to load documents:', err)
    } finally {
      setLoading(false)
    }
  }

  async function loadSignatures(docId: string) {
    try {
      const result = await api.documents.get(docId) as { document: Document; signatures: Signature[] }
      setSignatures((prev) => ({ ...prev, [docId]: result.signatures || [] }))
    } catch (err) {
      console.error('Failed to load signatures:', err)
    }
  }

  async function handleCreateDocument() {
    if (!createData.document_title || !createData.document_type) {
      alert('Title and type are required')
      return
    }
    setCreating(true)
    try {
      await api.documents.create({
        document_title: createData.document_title,
        document_type: createData.document_type,
        file_url: createData.file_url || undefined,
        related_entity_type: entityType,
        related_entity_id: entityId,
      })
      setCreateData({ document_title: '', document_type: '', file_url: '' })
      setShowCreateForm(false)
      loadDocuments()
    } catch (err) {
      console.error('Failed to create document:', err)
      alert('Failed to create document')
    } finally {
      setCreating(false)
    }
  }

  async function handleRequestSignature() {
    if (!signerData.signerName || !signerData.signerEmail || !sigFormDocId) return
    setSendingSignature(true)
    try {
      await api.documents.requestSignature(sigFormDocId, signerData.signerName, signerData.signerEmail)
      setSigFormDocId(null)
      setSignerData({ signerName: '', signerEmail: '' })
      loadSignatures(sigFormDocId)
    } catch (err) {
      console.error('Failed to request signature:', err)
      alert('Failed to request signature')
    } finally {
      setSendingSignature(false)
    }
  }

  async function handleDeleteDocument(docId: string) {
    if (!confirm('Delete this document?')) return
    try {
      await api.documents.delete(docId)
      loadDocuments()
    } catch {
      alert('Failed to delete document')
    }
  }

  if (loading) {
    return (
      <div className="py-4 flex items-center gap-2">
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-gray-400" />
        <span className="text-sm text-gray-500">Loading documents...</span>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">Documents</h3>
        {canEdit && (
          <button
            onClick={() => setShowCreateForm(!showCreateForm)}
            className="text-xs font-medium px-3 py-1.5 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors"
          >
            {showCreateForm ? 'Cancel' : '+ Attach Document'}
          </button>
        )}
      </div>

      {/* Create Document Form */}
      {showCreateForm && canEdit && (
        <div className="bg-gray-50 border border-gray-200 rounded-xl p-4 space-y-3">
          <div>
            <label className="text-xs text-gray-600 mb-1 block">Title</label>
            <input
              type="text"
              value={createData.document_title}
              onChange={(e) => setCreateData({ ...createData, document_title: e.target.value })}
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
              placeholder="e.g. Signed Lease Agreement"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-gray-600 mb-1 block">Type</label>
              <select
                value={createData.document_type}
                onChange={(e) => setCreateData({ ...createData, document_type: e.target.value })}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
              >
                <option value="">Select type...</option>
                {DOC_TYPES.map((t) => (
                  <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-xs text-gray-600 mb-1 block">File URL (optional)</label>
              <input
                type="url"
                value={createData.file_url}
                onChange={(e) => setCreateData({ ...createData, file_url: e.target.value })}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
                placeholder="https://..."
              />
            </div>
          </div>
          <button
            onClick={handleCreateDocument}
            disabled={creating}
            className="w-full py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-50"
          >
            {creating ? 'Attaching...' : 'Attach Document'}
          </button>
        </div>
      )}

      {/* Documents list */}
      {documents.length === 0 ? (
        <div className="text-center py-8 text-sm text-gray-400 bg-gray-50 rounded-xl border border-dashed border-gray-200">
          No documents attached yet
        </div>
      ) : (
        <div className="space-y-2">
          {documents.map((doc) => (
            <div key={doc.id} className="border border-gray-200 rounded-xl overflow-hidden">
              {/* Row */}
              <div
                className="flex items-center gap-3 px-4 py-3 bg-white hover:bg-gray-50 cursor-pointer transition-colors"
                onClick={() => {
                  const next = expandedDoc === doc.id ? null : doc.id
                  setExpandedDoc(next)
                  if (next && !signatures[doc.id]) loadSignatures(doc.id)
                }}
              >
                <svg className="w-4 h-4 text-gray-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">{doc.document_title}</p>
                  <p className="text-xs text-gray-500">{doc.document_type}</p>
                </div>
                <span className={`text-xs font-medium px-2 py-0.5 rounded-full shrink-0 ${
                  doc.signature_status === 'signed'
                    ? 'bg-green-100 text-green-700'
                    : 'bg-amber-100 text-amber-700'
                }`}>
                  {doc.signature_status === 'signed' ? '✓ Signed' : '○ Pending'}
                </span>
                <svg
                  className={`w-4 h-4 text-gray-400 shrink-0 transition-transform ${expandedDoc === doc.id ? 'rotate-180' : ''}`}
                  fill="none" viewBox="0 0 24 24" stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>

              {/* Expanded panel */}
              {expandedDoc === doc.id && (
                <div className="border-t border-gray-100 bg-gray-50 px-4 py-4 space-y-4">
                  {/* Signatures */}
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <p className="text-xs font-semibold text-gray-600 uppercase tracking-wide">Signatures</p>
                      {canEdit && sigFormDocId !== doc.id && (
                        <button
                          onClick={() => setSigFormDocId(doc.id)}
                          className="text-xs text-blue-600 hover:underline font-medium"
                        >
                          + Request Signature
                        </button>
                      )}
                    </div>

                    {sigFormDocId === doc.id && (
                      <div className="bg-white border border-blue-100 rounded-lg p-3 mb-2 space-y-2">
                        <div className="grid grid-cols-2 gap-2">
                          <input
                            type="text"
                            placeholder="Signer name"
                            value={signerData.signerName}
                            onChange={(e) => setSignerData({ ...signerData, signerName: e.target.value })}
                            className="px-2 py-1.5 border border-gray-200 rounded text-xs focus:outline-none focus:ring-1 focus:ring-blue-400"
                          />
                          <input
                            type="email"
                            placeholder="email@example.com"
                            value={signerData.signerEmail}
                            onChange={(e) => setSignerData({ ...signerData, signerEmail: e.target.value })}
                            className="px-2 py-1.5 border border-gray-200 rounded text-xs focus:outline-none focus:ring-1 focus:ring-blue-400"
                          />
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={handleRequestSignature}
                            disabled={sendingSignature}
                            className="px-3 py-1 bg-blue-600 text-white rounded text-xs font-medium hover:bg-blue-700 disabled:opacity-50"
                          >
                            {sendingSignature ? 'Sending...' : 'Send Request'}
                          </button>
                          <button
                            onClick={() => setSigFormDocId(null)}
                            className="px-3 py-1 text-gray-500 hover:text-gray-700 text-xs"
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    )}

                    {signatures[doc.id] && signatures[doc.id].length > 0 ? (
                      <div className="space-y-1.5">
                        {signatures[doc.id].map((sig) => (
                          <div key={sig.id} className="flex items-center justify-between bg-white border border-gray-100 rounded-lg px-3 py-2">
                            <div>
                              <p className="text-xs font-medium text-gray-900">{sig.signer_name}</p>
                              <p className="text-xs text-gray-500">{sig.signer_email}</p>
                            </div>
                                <div className="flex items-center gap-2">
                              <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                                sig.signature_status === 'signed'
                                  ? 'bg-green-100 text-green-700'
                                  : 'bg-yellow-100 text-yellow-700'
                              }`}>
                                {sig.signature_status === 'signed' ? '✓ Signed' : '○ Pending'}
                              </span>
                              {sig.signature_status !== 'signed' && (
                                <button
                                  title="Copy signature link"
                                  onClick={() => {
                                    const link = `${window.location.origin}/sign/${sig.id}`
                                    navigator.clipboard.writeText(link).then(() => alert('Link copied!'))
                                  }}
                                  className="text-xs text-gray-400 hover:text-gray-600"
                                >
                                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                  </svg>
                                </button>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-xs text-gray-400">No signature requests yet</p>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-3 pt-1">
                    {doc.file_url && (
                      <a
                        href={doc.file_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-blue-600 hover:underline font-medium flex items-center gap-1"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                        </svg>
                        View File
                      </a>
                    )}
                    {canEdit && (
                      <button
                        onClick={() => handleDeleteDocument(doc.id)}
                        className="text-xs text-red-500 hover:underline font-medium ml-auto"
                      >
                        Delete
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
