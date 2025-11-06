export function sortByPriority<T extends {priority:number}>(items:T[]){return items.slice().sort((a,b)=>a.priority-b.priority)}
