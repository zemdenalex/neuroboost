import React from 'react'
export default function Modal({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black/50">
      <div className="bg-zinc-900 border border-zinc-700 rounded p-6 relative">
        <button className="absolute top-2 right-2 text-zinc-400" onClick={onClose}>×</button>
        {children}
      </div>
    </div>
  )
}
