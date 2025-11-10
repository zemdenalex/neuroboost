import React from 'react'
export default function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={`px-3 py-2 bg-zinc-900 border border-zinc-700 rounded text-sm text-zinc-100 ${props.className??''}`} />
}
