import { NextRequest, NextResponse } from 'next/server'

export async function POST(req: NextRequest) {
  const { token } = await req.json()

  if (!token) {
    return NextResponse.json({ success: false, error: 'Missing token' }, { status: 400 })
  }

  const secret = process.env.TURNSTILE_SECRET_KEY
  if (!secret) {
    return NextResponse.json({ success: false, error: 'Turnstile not configured' }, { status: 500 })
  }

  const formData = new FormData()
  formData.append('secret', secret)
  formData.append('response', token)
  formData.append('remoteip', req.headers.get('x-forwarded-for') ?? req.headers.get('x-real-ip') ?? '')

  const cfRes = await fetch('https://challenges.cloudflare.com/turnstile/v0/siteverify', {
    method: 'POST',
    body: formData,
  })

  const data = await cfRes.json()
  if (!data.success) {
    return NextResponse.json({ success: false, error: 'Turnstile verification failed' }, { status: 400 })
  }

  return NextResponse.json({ success: true })
}
