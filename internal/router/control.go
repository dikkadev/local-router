package router

const controlHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>local-router</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;line-height:1.4;color:#e8e8e8;background:#0a0a0a}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #333;padding:.45rem;text-align:left}code{background:#1a1a1a;color:#f0f0f0;padding:.1rem .25rem}button{cursor:pointer;color:#e8e8e8;background:#1a1a1a;border:1px solid #555}button:hover:not(:disabled){background:#242424}button:active:not(:disabled){background:#111}button:disabled{cursor:wait;opacity:.65}button:focus-visible{outline:2px solid #8ab4f8;outline-offset:2px}a{color:#8ab4f8}h1{display:flex;align-items:center;gap:1rem}.icon-button{display:inline-grid;place-items:center;width:1.75rem;height:1.75rem;padding:0;font-size:1rem;line-height:1}.icon-button svg{display:block;width:1em;height:1em}.spinning{animation:spin .35s linear}@keyframes spin{to{transform:rotate(360deg)}}.muted{color:#aaa}.notice{min-height:1.4em}.notice.error{color:#ff8a8a}.notice.success{color:#9be28f}
</style>
</head>
<body>
<h1><span>local-router</span><button id="reload" class="icon-button" type="button" title="Reload routes" aria-label="Reload routes"><svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M17.7 6.3A8 8 0 1 0 20 12h-2a6 6 0 1 1-1.8-4.3L13 11h8V3z"/></svg></button></h1>
<p class="muted">Stable local routes. API root: <code>/router/routes</code>.</p>
<p id="status" class="notice" role="status" aria-live="polite"></p>
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
      tr.innerHTML='<td>'+esc(route.name)+'</td><td><a href="'+route.url+'" target="_blank" rel="noopener noreferrer">'+route.url+'</a></td><td>'+esc(route.title||'')+'</td><td>'+esc(route.status)+'</td><td>'+esc(target)+'</td><td>'+esc(route.heartbeatPath)+'</td><td>'+route.misses+'</td><td>'+esc(route.exec||'')+'</td><td><button data-open="'+route.url+'">Open</button> <button data-copy="'+route.url+'">Copy</button> <button data-delete="'+route.name+'">Unregister</button> <button data-pin="'+route.name+'" data-pinned="'+route.pinned+'">'+(route.pinned?'Unpin':'Pin')+'</button> <button data-kill="'+route.name+'">Kill</button></td>';
      return tr;
    }));
  }catch(err){body.innerHTML='<tr><td colspan="9">Failed to load routes: '+esc(err.message)+'</td></tr>';}
}
function esc(value){return String(value).replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));}
function setStatus(message,type){const el=document.getElementById('status'); el.textContent=message||''; el.className='notice '+(type||'');}
async function readResponseError(res){try{const data=await res.json(); return data.error||JSON.stringify(data);}catch{return await res.text()||res.statusText;}}
async function checkedFetch(url,options){const res=await fetch(url,options); if(!res.ok) throw new Error(await readResponseError(res)); return res;}
document.addEventListener('click',async event=>{
  const b=event.target.closest('button'); if(!b)return;
  try{
    if(b.id==='reload'){
      const icon=b.querySelector('svg'); icon.classList.remove('spinning'); void icon.offsetWidth; icon.classList.add('spinning'); icon.addEventListener('animationend',()=>icon.classList.remove('spinning'),{once:true}); await loadRoutes();
    }
    if(b.dataset.open) window.open(b.dataset.open,'_blank','noopener,noreferrer');
    if(b.dataset.copy){await navigator.clipboard.writeText(b.dataset.copy); setStatus('Copied '+b.dataset.copy,'success');}
    if(b.dataset.delete){await checkedFetch('/router/routes/'+b.dataset.delete,{method:'DELETE'}); setStatus('Unregistered '+b.dataset.delete,'success'); await loadRoutes();}
    if(b.dataset.pin){await checkedFetch('/router/routes/'+b.dataset.pin,{method:'PATCH',headers:{'content-type':'application/json'},body:JSON.stringify({pinned:b.dataset.pinned!=='true'})}); setStatus((b.dataset.pinned==='true'?'Unpinned ':'Pinned ')+b.dataset.pin,'success'); await loadRoutes();}
    if(b.dataset.kill){
      const label=b.textContent; b.disabled=true; b.textContent='Killing…'; setStatus('Killing '+b.dataset.kill+'…','');
      const res=await checkedFetch('/router/routes/'+b.dataset.kill+'/kill',{method:'POST'});
      const data=await res.json();
      setStatus('Killed '+b.dataset.kill+(data.pids&&data.pids.length?' (PID '+data.pids.join(', ')+')':''),'success');
      b.textContent=label; b.disabled=false; await loadRoutes();
    }
  }catch(err){setStatus(err.message,'error'); if(b.disabled){b.disabled=false; b.textContent='Kill';}}
});
loadRoutes(); setInterval(loadRoutes,5000);
</script>
</body>
</html>`
