package router

const controlHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>local-router</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;line-height:1.4;color:#e8e8e8;background:#0a0a0a}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #333;padding:.45rem;text-align:left}tr.pinned{background:rgba(244,199,80,.055);box-shadow:inset 3px 0 0 #d8b84b}tr.pinned td{border-bottom-color:#3e3620}tr.missed{background:rgba(255,109,64,.075);box-shadow:inset 3px 0 0 #f06d3d}tr.missed td{border-bottom-color:#4a281f}code{background:#1a1a1a;color:#f0f0f0;padding:.1rem .25rem}button{cursor:pointer;color:#e8e8e8;background:#1a1a1a;border:1px solid #555;border-radius:.25rem}button:hover:not(:disabled){background:#242424;border-color:#777}button:active:not(:disabled){background:#111}button:disabled{cursor:wait;opacity:.65}button:focus-visible,input:focus-visible{outline:2px solid #8ab4f8;outline-offset:2px}a{color:#8ab4f8}h1{display:flex;align-items:center;gap:.6rem;margin-bottom:.25rem}.icon-button{display:inline-grid;place-items:center;width:1.75rem;height:1.75rem;padding:0;font-size:1rem;line-height:1}.icon-button svg{display:block;width:1em;height:1em}.spinning{animation:spin .35s linear}@keyframes spin{to{transform:rotate(360deg)}}.muted{color:#aaa}.notice{min-height:1.4em}.notice.error{color:#ff8a8a}.notice.success{color:#9be28f}.age{font-weight:600}.datetime{display:block;color:#aaa;font-size:.85em;line-height:1.2}.route-name{font-weight:650}.pin-mark,.miss-mark{display:inline-flex;align-items:center;margin-left:.35rem;font-size:.8rem;letter-spacing:.02em}.pin-mark{color:#f4d16c}.miss-mark{color:#ff8a5c}.status-badge{display:inline-flex;align-items:center;border:1px solid #444;border-radius:999px;padding:.05rem .45rem;font-size:.85em}.status-badge.pinned{border-color:#7b672e;color:#f4d16c;background:rgba(244,199,80,.08)}.status-badge.missed{border-color:#8a3f2b;color:#ff9a70;background:rgba(255,109,64,.1)}dialog{width:min(30rem,calc(100vw - 2rem));padding:0;border:1px solid #383838;border-radius:.8rem;background:#101010;color:#e8e8e8;box-shadow:0 1.5rem 5rem rgba(0,0,0,.65)}dialog::backdrop{background:rgba(0,0,0,.62);backdrop-filter:blur(2px)}.modal{padding:1rem}.modal header{display:flex;align-items:center;justify-content:space-between;gap:1rem;margin-bottom:.35rem}.modal h2{margin:0;font-size:1.15rem}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:.8rem}.form-grid label{display:flex;flex-direction:column;gap:.25rem;font-size:.9rem;color:#cfcfcf}.form-grid .full{grid-column:1/-1}.form-grid input{box-sizing:border-box;width:100%;padding:.45rem .5rem;border:1px solid #454545;border-radius:.35rem;color:#f0f0f0;background:#090909}.form-grid input::placeholder{color:#666}.checkbox-row{flex-direction:row!important;align-items:center;gap:.45rem}.checkbox-row input{width:auto}.modal-actions{display:flex;justify-content:flex-end;gap:.5rem;margin-top:1rem}.route-actions{white-space:nowrap}.route-actions button{margin:.1rem .05rem;padding:.18rem .4rem}
</style>
</head>
<body>
<h1><span>local-router</span><button id="reload" class="icon-button" type="button" title="Reload routes" aria-label="Reload routes"><svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M17.7 6.3A8 8 0 1 0 20 12h-2a6 6 0 1 1-1.8-4.3L13 11h8V3z"/></svg></button><button id="add-route" class="icon-button" type="button" title="Register route" aria-label="Register route"><svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6z"/></svg></button></h1>
<p class="muted">Stable local routes. API root: <code>/router/routes</code>.</p>
<p id="status" class="notice" role="status" aria-live="polite"></p>
<table>
<thead><tr><th>Name</th><th>URL</th><th>Title</th><th>Status</th><th>Age</th><th>Registered</th><th>Target</th><th>Heartbeat</th><th>Misses</th><th>Exec</th><th>Actions</th></tr></thead>
<tbody id="routes"><tr><td colspan="11">Loading…</td></tr></tbody>
</table>
<dialog id="register-dialog" aria-labelledby="register-title">
<form id="register-form" class="modal">
<header><h2 id="register-title">Register route</h2><button id="close-register" class="icon-button" type="button" aria-label="Close register dialog">×</button></header>
<p class="muted">Point a friendly <code>.localhost</code> name at an already-running local server.</p>
<div class="form-grid">
<label>Name<input id="route-name" name="name" required autocomplete="off" placeholder="demo app"></label>
<label>Port<input id="route-port" name="port" type="number" min="1" max="65535" required inputmode="numeric" placeholder="5173"></label>
<label class="full">Title<input id="route-title" name="title" autocomplete="off" placeholder="Demo app"></label>
<label>Target host<input id="route-target-host" name="targetHost" autocomplete="off" placeholder="127.0.0.1"></label>
<label>Heartbeat path<input id="route-heartbeat-path" name="heartbeatPath" autocomplete="off" placeholder="/"></label>
<label class="checkbox-row"><input id="route-pinned" name="pinned" type="checkbox"> Pin this route</label>
<label class="checkbox-row"><input id="route-force" name="force" type="checkbox"> Replace existing route</label>
</div>
<p id="register-error" class="notice error" role="alert"></p>
<footer class="modal-actions"><button id="cancel-register" type="button">Cancel</button><button id="submit-register" type="submit">Register</button></footer>
</form>
</dialog>
<script>
const registerDialog=document.getElementById('register-dialog');
const registerForm=document.getElementById('register-form');
async function loadRoutes(){
  const body=document.getElementById('routes');
  try{
    const res=await fetch('/router/routes',{cache:'no-store'});
    const routes=await res.json();
    if(!routes.length){body.innerHTML='<tr><td colspan="11" class="muted">No routes registered.</td></tr>';return;}
    body.replaceChildren(...routes.map(route=>{
      const tr=document.createElement('tr');
      const hasMisses=Number(route.misses)>0;
      const classes=[];
      if(route.pinned) classes.push('pinned');
      if(hasMisses) classes.push('missed');
      tr.className=classes.join(' ');
      const target=route.targetHost+':'+route.port;
      const registered=formatRegistered(route.createdAt);
      const pinMark=route.pinned?'<span class="pin-mark" title="Pinned">pinned</span>':'';
      const missMark=hasMisses?'<span class="miss-mark" title="Heartbeat misses">'+route.misses+' miss'+(route.misses===1?'':'es')+'</span>':'';
      const nameCell='<span class="route-name">'+esc(route.name)+'</span>'+pinMark+missMark;
      const displayURL=urlForRoute(route);
      const statusClasses=['status-badge'];
      if(route.pinned) statusClasses.push('pinned');
      if(hasMisses) statusClasses.push('missed');
      tr.innerHTML='<td>'+nameCell+'</td><td><a href="'+displayURL+'" target="_blank" rel="noopener noreferrer">'+displayURL+'</a></td><td>'+esc(route.title||'')+'</td><td><span class="'+statusClasses.join(' ')+'">'+esc(route.status)+'</span></td><td><span class="age">'+esc(registered.age)+'</span></td><td><span class="datetime">'+esc(registered.date)+'<br>'+esc(registered.time)+'</span></td><td>'+esc(target)+'</td><td>'+esc(route.heartbeatPath)+'</td><td>'+route.misses+'</td><td>'+esc(route.exec||'')+'</td><td class="route-actions"><button data-open="'+displayURL+'">Open</button> <button data-copy="'+displayURL+'">Copy</button> <button data-delete="'+route.name+'">Unregister</button> <button data-pin="'+route.name+'" data-pinned="'+route.pinned+'">'+(route.pinned?'Unpin':'Pin')+'</button> <button data-kill="'+route.name+'">Kill</button></td>';
      return tr;
    }));
  }catch(err){body.innerHTML='<tr><td colspan="11">Failed to load routes: '+esc(err.message)+'</td></tr>';}
}
function esc(value){return String(value).replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));}
function urlForRoute(route){
  if(location.hostname==='dev.localhost'||location.hostname==='router.localhost') return route.url;
  const port=location.port?':'+location.port:'';
  return location.protocol+'//'+route.name+'.'+location.hostname+port;
}
function formatRegistered(value){
  const date=new Date(value);
  if(Number.isNaN(date.getTime())) return {age:'unknown',date:'',time:''};
  const seconds=Math.max(0,Math.floor((Date.now()-date.getTime())/1000));
  const mins=Math.floor(seconds/60), hours=Math.floor(mins/60), days=Math.floor(hours/24);
  let age;
  if(seconds<60) age=seconds+'s';
  else if(mins<60) age=mins+'m'+String(seconds%60).padStart(2,'0')+'s';
  else if(hours<24) age=hours+'h'+String(mins%60).padStart(2,'0')+'min';
  else age=days+'d'+(hours%24?String(hours%24)+'h':'');
  return {
    age,
    date:date.toLocaleDateString(undefined,{month:'short',day:'numeric'}),
    time:date.toLocaleTimeString(undefined,{hour:'2-digit',minute:'2-digit'})
  };
}
function setStatus(message,type){const el=document.getElementById('status'); el.textContent=message||''; el.className='notice '+(type||'');}
function setRegisterError(message){document.getElementById('register-error').textContent=message||'';}
function openRegisterDialog(){
  registerForm.reset();
  setRegisterError('');
  if(registerDialog.showModal) registerDialog.showModal(); else registerDialog.setAttribute('open','');
  document.getElementById('route-name').focus();
}
function closeRegisterDialog(){registerDialog.close?registerDialog.close():registerDialog.removeAttribute('open');}
async function readResponseError(res){try{const data=await res.json(); return data.error||JSON.stringify(data);}catch{return await res.text()||res.statusText;}}
async function checkedFetch(url,options){const res=await fetch(url,options); if(!res.ok) throw new Error(await readResponseError(res)); return res;}
document.addEventListener('click',async event=>{
  const b=event.target.closest('button'); if(!b)return;
  if(b.id==='add-route'){openRegisterDialog(); return;}
  if(b.id==='close-register'||b.id==='cancel-register'){closeRegisterDialog(); return;}
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
registerForm.addEventListener('submit',async event=>{
  event.preventDefault();
  const data=new FormData(registerForm);
  const name=String(data.get('name')||'').trim();
  const port=Number(data.get('port'));
  if(!name){setRegisterError('Name is required.'); return;}
  if(!Number.isInteger(port)||port<1||port>65535){setRegisterError('Port must be between 1 and 65535.'); return;}
  const payload={port};
  for(const key of ['title','targetHost','heartbeatPath']){
    const value=String(data.get(key)||'').trim();
    if(value) payload[key]=value;
  }
  if(data.get('pinned')) payload.pinned=true;
  const submit=document.getElementById('submit-register');
  submit.disabled=true; submit.textContent='Registering…'; setRegisterError('');
  try{
    const force=data.get('force')?'?force=true':'';
    const res=await checkedFetch('/router/routes/'+encodeURIComponent(name)+force,{method:'PUT',headers:{'content-type':'application/json'},body:JSON.stringify(payload)});
    const route=await res.json();
    closeRegisterDialog();
    setStatus('Registered '+route.name+' at '+urlForRoute(route),'success');
    await loadRoutes();
  }catch(err){setRegisterError(err.message);}
  finally{submit.disabled=false; submit.textContent='Register';}
});
loadRoutes(); setInterval(loadRoutes,5000);
</script>
</body>
</html>`
