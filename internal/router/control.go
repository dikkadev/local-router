package router

const controlHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>local-router</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;line-height:1.4;color:#e8e8e8;background:#0a0a0a}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #333;padding:.45rem;text-align:left}code{background:#1a1a1a;color:#f0f0f0;padding:.1rem .25rem}button{cursor:pointer;color:#e8e8e8;background:#1a1a1a;border:1px solid #555}a{color:#8ab4f8}.muted{color:#aaa}
</style>
</head>
<body>
<h1>local-router</h1>
<p class="muted">Stable local routes. API root: <code>/router/routes</code>.</p>
<table>
<thead><tr><th>Name</th><th>URL</th><th>Title</th><th>Status</th><th>Target</th><th>Heartbeat</th><th>Misses</th><th>Exec</th><th>Actions</th></tr></thead>
<tbody id="routes"><tr><td colspan="9">Loading…</td></tr></tbody>
</table>
<script>
async function loadRoutes(){
  const body=document.getElementById('routes');
  try{
    const res=await fetch('/router/routes',{cache:'no-store'});
    const routes=await res.json();
    if(!routes.length){body.innerHTML='<tr><td colspan="9" class="muted">No routes registered.</td></tr>';return;}
    body.replaceChildren(...routes.map(route=>{
      const tr=document.createElement('tr');
      const target=route.targetHost+':'+route.port;
      tr.innerHTML='<td>'+esc(route.name)+'</td><td><a href="'+route.url+'" target="_blank" rel="noopener noreferrer">'+route.url+'</a></td><td>'+esc(route.title||'')+'</td><td>'+esc(route.status)+'</td><td>'+esc(target)+'</td><td>'+esc(route.heartbeatPath)+'</td><td>'+route.misses+'</td><td>'+esc(route.exec||'')+'</td><td><button data-open="'+route.url+'">Open</button> <button data-copy="'+route.url+'">Copy</button> <button data-delete="'+route.name+'">Unregister</button> <button data-pin="'+route.name+'" data-pinned="'+route.pinned+'">'+(route.pinned?'Unpin':'Pin')+'</button></td>';
      return tr;
    }));
  }catch(err){body.innerHTML='<tr><td colspan="9">Failed to load routes: '+esc(err.message)+'</td></tr>';}
}
function esc(value){return String(value).replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));}
document.addEventListener('click',async event=>{
  const b=event.target.closest('button'); if(!b)return;
  if(b.dataset.open) window.open(b.dataset.open,'_blank','noopener,noreferrer');
  if(b.dataset.copy) await navigator.clipboard.writeText(b.dataset.copy);
  if(b.dataset.delete){await fetch('/router/routes/'+b.dataset.delete,{method:'DELETE'}); await loadRoutes();}
  if(b.dataset.pin){await fetch('/router/routes/'+b.dataset.pin,{method:'PATCH',headers:{'content-type':'application/json'},body:JSON.stringify({pinned:b.dataset.pinned!=='true'})}); await loadRoutes();}
});
loadRoutes(); setInterval(loadRoutes,5000);
</script>
</body>
</html>`
