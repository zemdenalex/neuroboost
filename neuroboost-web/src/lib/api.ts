const BASE_URL = import.meta.env.VITE_API_URL ?? '/api'
async function request(method: string, url: string, body?: any) {
  const full = url.startsWith('/')
    ? `${BASE_URL}${url}`
    : `${BASE_URL}/${url.replace(/^\//,'')}`
  console.log('[API STUB]', method, full, body ?? null)
  // Return a stub structure
  return { data: null }
}
export async function get(url: string) { return request('GET', url) }
export async function post(url: string, body: any) { return request('POST', url, body) }
export async function patch(url: string, body: any) { return request('PATCH', url, body) }
export async function del(url: string) { return request('DELETE', url) }
