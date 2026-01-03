import React from 'react'
import Header from './Header'

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-black text-zinc-100 font-mono">
      <Header />
      <main className="pt-16">{children}</main>
    </div>
  )
}
