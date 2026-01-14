import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Toaster } from '@/components/ui/toaster'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'URL Shortener',
  description: 'Create and manage short URLs with analytics',
}

import { Auth0Provider } from '@auth0/nextjs-auth0/client';

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <Auth0Provider>
        <body className={inter.className}>
          {children}
          <Toaster />
        </body>
      </Auth0Provider>
    </html>
  )
}
