import React from 'react'
export default function Select({children, className='', ...rest}: React.SelectHTMLAttributes<HTMLSelectElement> & {children:React.ReactNode}) {
  return <select className={`px-3 py-2 bg-zinc-900 border border-zinc-700 rounded text-sm text-zinc-100 ${className}`} {...rest}>{children}</select>
}
