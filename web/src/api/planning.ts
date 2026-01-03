import { api } from './client'
export async function getGraph() { console.log('[API STUB] get planning graph'); return api.get('/planning/graph') }
export async function createNode(node: any) { console.log('[API STUB] create planning node', node); return api.post('/planning/nodes', node) }
export async function createEdge(edge: any) { console.log('[API STUB] create planning edge', edge); return api.post('/planning/edges', edge) }
