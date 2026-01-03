import React from 'react'
export default function Button({children, className='', ...rest}: React.ButtonHTMLAttributes<HTMLButtonElement> & {children:React.ReactNode}) {
  return <button className={`px-4 py-2 bg-zinc-800 rounded ${className}`} {...rest}>{children}</button>
}
