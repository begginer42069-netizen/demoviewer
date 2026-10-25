const $=id=>document.getElementById(id);
const css=v=>getComputedStyle(document.documentElement).getPropertyValue(v).trim();
const mmss=s=>s==null?'—':Math.floor(s/60)+':'+String(Math.floor(s%60)).padStart(2,'0');
const fx=(v,d=3)=>v==null||isNaN(v)?'—':v.toFixed(d);
let STATE=null,J=null,WI=null,sel=null,SORT={k:'dist',d:1};
try{const s=JSON.parse(localStorage.getItem('dvSort')||'null');if(s&&s.k)SORT=s}catch(e){}

async function loadState(){const r=await fetch('/api/state');STATE=await r.json();$('folder').value=STATE.folder||'';$('mapsdir').value=STATE.mapsFolder||'';renderRank()}
async function browse(kind){const r=await fetch('/api/browse?kind='+kind);if(!r.ok){alertBox(await r.text());return}const x=await r.json();if(x.cancelled)return;
 await loadState();if(kind==='demo'&&x.file){const hs=STATE.jumps.filter(j=>j.name===x.file||j.name.endsWith('\\'+x.file)||j.name.endsWith('/'+x.file)),hit=hs.find(j=>j.main)||hs[0];if(hit)openJump(hit.d,hit.j);else alertBox('No long jumps found in '+x.file)}
 else if(J&&kind==='maps'){openJump(J._d,J._j)}}
$('bfolder').onclick=()=>browse('folder');$('bdemo').onclick=()=>browse('demo');$('bmaps').onclick=()=>browse('maps');
function renderRank(){
 const types=[...new Set(STATE.jumps.map(x=>x.type))].sort(),tf0=$('tfilt').value;
 $('tfilt').innerHTML='<option value="all">all types</option>'+types.map(x=>`<option${x===tf0?' selected':''}>${x}</option>`).join('');
 const f=$('filt').value,tf=$('tfilt').value,rows=STATE.jumps.filter(x=>(f==='all'||(f==='main'?x.main:x.result===f))&&(tf==='all'||x.type===tf));
 $('cnt').textContent=`${STATE.demos.length} demos, ${STATE.jumps.filter(x=>x.main).length} best jumps, ${STATE.jumps.length} attempts`+(STATE.demos.some(d=>d.Error)?` (${STATE.demos.filter(d=>d.Error).length} unreadable)`:'');
 if(!STATE.jumps.length){$('rank').innerHTML=`<tr><td>No long jumps found in <b>${STATE.folder}</b> or its subfolders. Type another folder above.</td></tr>`;return}
 const COLS=[['#',null],['demo','name','t'],['time','time'],['map','map','t'],['type','type','t'],['result','result','t'],['distance','dist'],['pre','pre'],['jumpoff','jumpoff'],['block','block'],['landing','landEdge'],['strafes','strafes'],['eff','eff'],['mouse slaps','slapScore']];
 if(SORT.k){const c=COLS.find(c=>c[1]===SORT.k),txt=c&&c[2]==='t',d=SORT.d;
  rows.sort((p,q)=>{const a=p[SORT.k],b=q[SORT.k];if(a==null||(typeof a==='number'&&isNaN(a)))return 1;if(b==null||(typeof b==='number'&&isNaN(b)))return -1;
   if(txt)return d*String(a).localeCompare(String(b),undefined,{numeric:true,sensitivity:'base'})||((p.time||0)-(q.time||0));
   return d*(b-a)||(q.dist-p.dist)})}
 $('rank').innerHTML='<tr>'+COLS.map(c=>{const on=c[1]&&SORT.k===c[1],arr=on?(SORT.d===1?(c[2]==='t'?' ▲':' ▼'):(c[2]==='t'?' ▼':' ▲')):'';
   const tip=c[0]==='mouse slaps'?' title="strafe switches where the mouse hits the other side like a wall (fast on both sides of the flip)"':'';
   return `<th${tip} ${c[1]?`data-s="${c[1]}" class="srt${on?' on':''}"`:''}>${c[0]}${arr}</th>`}).join('')+'</tr>'+
 rows.map((x,i)=>`<tr data-k="${x.d}_${x.j}" class="${sel===x.d+'_'+x.j?'sel':''}"><td>${i+1}</td><td>${x.name}</td><td class="mute">${mmss(x.time)}</td><td>${x.map}</td><td>${x.type}${x.style?' <span class="mute">'+x.style+'</span>':''}</td><td class="${x.result==='LANDED'?'ok':'bad'}">${x.result.toLowerCase()}</td><td><b>${fx(x.dist)}</b></td><td>${fx(x.pre)}</td><td>${fx(x.jumpoff)}</td><td>${x.block==null?'—':x.block.toFixed(0)}</td><td>${fx(x.landEdge)}</td><td>${x.strafes}</td><td>${x.eff.toFixed(0)}%</td><td>${x.oneFrameSlaps}/${Math.max(x.slaps-1,0)} <span class="mute">· ${(x.slapScore||0).toFixed(0)}%</span></td></tr>`).join('');
 document.querySelectorAll('#rank tr[data-k]').forEach(tr=>tr.onclick=()=>openJump(...tr.dataset.k.split('_').map(Number)));
 document.querySelectorAll('#rank th[data-s]').forEach(th=>th.onclick=()=>{const k=th.dataset.s;SORT=SORT.k===k?{k,d:-SORT.d}:{k,d:1};
  try{localStorage.setItem('dvSort',JSON.stringify(SORT))}catch(e){}renderRank()});
}
$('filt').onchange=renderRank;$('tfilt').onchange=renderRank;
$('open').onclick=async()=>{const r=await fetch('/api/folder',{method:'POST',body:JSON.stringify({Path:$('folder').value})});if(!r.ok){alertBox(await r.text());return}await loadState()};
function alertBox(t){$('cnt').textContent=t}

async function openJump(d,j){
 sel=d+'_'+j;renderRank();
 const r=await fetch(`/api/jump?d=${d}&j=${j}`);J=await r.json();J._d=d;J._j=j;WI=null;
 J.plugin=J.plugin||[];J.strafes=J.strafes||[];J.fogBase=J.fogBase||[];J.chart=J.chart||[];
 $('detail').style.display='block';
 const sty={bw:' (backwards)',sw:' (sideways)'}[J.style]||'';const nm=J.pluginType?`${J.type}${sty} <span class="mute" style="font-weight:400">(plugin: ${J.pluginType})</span>`:J.type+sty;
 $('title').innerHTML=`${J.demo} &mdash; ${nm} <span class="${J.landed?'ok':'bad'}">${J.result.toLowerCase()}</span>`;
 const st=(k,v,s)=>`<div class="stat"><b>${v}</b><span>${k}${s?' &middot; '+s:''}</span></div>`;
 const le=J.landEdge==null?'—':(J.landEdge>=0?'+':'')+J.landEdge.toFixed(3);
 $('stats').innerHTML=st('distance',fx(J.dist),J.landed?'landed':'failed')+st('prestrafe',fx(J.pre),J.fog+' FOG')+(J.fog>1&&J.fogBase&&J.fogBase.length?st('speed on landing',fx(J.fogBase[0].pre),'before '+(J.fog-1)+' ground frame'+(J.fog>2?'s':'')+' of friction'):'')+st('jumpoff',fx(J.jumpoff),J.mapFound?'':'map not found')+
  st(J.landed?'landing':'lasting',le,J.landed?'reserve':'units short')+st('block',J.block==null?'—':J.block.toFixed(0),J.block==null&&J.mapFound?'not in map file':'')+
  st('maxspeed',fx(J.max,1),'+'+J.gainTotal.toFixed(1))+st('strafes',J.strafes.length,'eff '+J.eff.toFixed(0)+'%')+st('height',fx(J.height,1))+st('veer',fx(J.veer,2),'sideways drift')+
  st('air time',J.airFrames+' frames')+st('dead airtime',(J.deadAir||0)+' frames','no strafe key held')+(()=>{const ya=yawAccStats();return ya?st('yaw accel at switches',ya.avg.toFixed(2)+'°','avg per switch · max '+ya.max.toFixed(2)+'°/frame²'):''})()+st('mouse slaps',J.oneFrameSlaps+' / '+Math.max(J.slaps-1,0),'avg 2-frame swing '+(J.slapScore||0).toFixed(0)+'% (slap ≥ 110%)')+cvarBoxes();
 $('strafes').innerHTML='<tr><th>#</th><th>key</th><th>gain</th><th>loss</th><th>frames</th><th>eff</th><th>overlaps</th><th title="dead airtime: frames in the air with no strafe key (W/A/S/D) held, counted after this strafe">dead air</th><th title="GC: mouse turn compared with the ideal turn for your speed. Over 100% = turning faster than ideal.">avg eff (max)</th></tr>'+J.strafes.map((s,i)=>`<tr><td>${i+1}</td><td>${s.key}</td><td>${s.gain.toFixed(1)}</td><td>${s.loss.toFixed(1)}</td><td>${s.frames}</td><td>${s.eff.toFixed(0)}%</td><td>${s.ovl}</td><td class="${s.dead>0?'bad':''}">${s.dead}</td><td>${s.avgEff<0?'—':s.avgEff.toFixed(0)+'% <span class="mute">('+s.maxEff.toFixed(0)+'%)</span>'}</td></tr>`).join('');
 $('plugin').innerHTML=J.plugin.length?J.plugin.map(p=>`<div class="mute small">${p.src}</div><pre class="pl">${p.text.replace(/[<&]/g,c=>c==='<'?'&lt;':'&amp;')}</pre>`).join('')+'<p class="mute small" style="margin:0">GC can read 0.4&ndash;0.8 lower than kz_jump_stats on fails where you hit the side of the block on the same frame you reach landing height.</p>':'<p class="mute">No jump stats were recorded in this demo for this jump.</p>';
 $('wpre').value=J.pre.toFixed(2);$('wedge').value=J.jumpoff==null?'':J.jumpoff.toFixed(2);$('wedge').disabled=J.jumpoff==null;
 $('wfog').max=Math.min(J.fog,30);$('wfog').value=J.fog;$('wfog').disabled=J.fog<=1;
 $('wres').innerHTML='';$('rmode').value='real';$('rmode').options[1].disabled=true;
 setReplay();drawCharts();tlReset();
}

async function runWI(pre,edge){
 if(!J)return;const fog=+$('wfog').value||J.fog;const q=`/api/whatif?d=${J._d}&j=${J._j}&pre=${pre}&fog=${fog}`+(edge!==''&&edge!=null?`&edge=${edge}`:'');
 const r=await fetch(q);if(!r.ok){$('wres').textContent=await r.text();return}WI=await r.json();WI.strafes=WI.strafes||[];WI.path=WI.path||[];
 const d=WI.dist-J.dist;
 let verdict=!WI.known?`<span class="mute">land/fail unknown (block not found in the map)</span>`:WI.landed?`<span class="ok">LANDS</span> <span class="small mute">by ${WI.landEdge.toFixed(2)} units</span>`:`<span class="bad">FAILS</span> <span class="small mute">${(-WI.landEdge).toFixed(2)} units short</span>`;
 const cap=WI.capped?`<p class="bad small">Take-off speed over 300 u/s: the CS 1.6 bunnyhop cap cuts it to ${WI.capSpeed.toFixed(1)} u/s, so the jump dies.</p>`:'';
 const near=WI.known&&Math.abs(WI.landEdge)<Math.max(0.3,Math.abs(WI.checkDiff)*2)?`<p class="small" style="color:var(--air)">Too close to call: the margin is within the simulator's error on this jump.</p>`:'';
 const rows=WI.strafes.map((s,i)=>{const o=J.strafes[i];return `<tr><td>${i+1}</td><td>${s.key}</td><td>${o?o.gain.toFixed(1):'—'}</td><td class="wi">${s.gain.toFixed(1)}</td><td>${s.eff.toFixed(0)}%</td></tr>`}).join('');
 $('wres').innerHTML=`<div class="verdict">${verdict}</div>${cap}${near}
 <div class="stats"><div class="stat"><b class="wi">${fx(WI.dist)}</b><span>distance &middot; ${(d>=0?'+':'')+d.toFixed(2)} vs real</span></div>
 <div class="stat"><b>${fx(WI.pre,2)}</b><span>prestrafe &middot; ${WI.fog} FOG</span></div><div class="stat"><b>${WI.edge==null?'—':fx(WI.edge,2)}</b><span>jumpoff</span></div>
 <div class="stat"><b>${fx(WI.max,1)}</b><span>maxspeed</span></div><div class="stat"><b>${WI.eff.toFixed(0)}%</b><span>efficiency</span></div>
 <div class="stat"><b>${(WI.checkDiff>=0?'+':'')+WI.checkDiff.toFixed(3)}</b><span>self-check: simulator vs the real jump</span></div></div>
 <table style="max-width:460px"><tr><th>#</th><th>key</th><th>real gain</th><th>what-if gain</th><th>what-if eff</th></tr>${rows}</table>
 <p class="mute small">Self-check: the same simulator re-running the real take-off lands ${Math.abs(WI.checkDiff).toFixed(3)} units ${WI.checkDiff<0?'short of':'past'} the real distance. Margins smaller than that are a coin flip.</p>`;
 $('rmode').options[1].disabled=false;drawCharts();
}
$('wrun').onclick=()=>runWI($('wpre').value,$('wedge').disabled?'':$('wedge').value);
function fogFill(){const n=Math.max(1,Math.min(+$('wfog').value||J.fog,J.fog));$('wfog').value=n;const fp=n<J.fog&&J.fogBase[n-1];
 if(fp){$('wpre').value=fp.pre.toFixed(2);if(fp.jumpoff!=null)$('wedge').value=fp.jumpoff.toFixed(2)}else{$('wpre').value=J.pre.toFixed(2);$('wedge').value=J.jumpoff==null?'':J.jumpoff.toFixed(2)}}
$('wfog').onchange=fogFill;
$('wreal').onclick=()=>{$('wfog').value=J.fog;$('wpre').value=J.pre.toFixed(2);$('wedge').value=J.jumpoff==null?'':J.jumpoff.toFixed(2);runWI($('wpre').value,$('wedge').value)};
$('rmode').onchange=()=>setReplay(true);

// ---------- charts ----------
function setup(c){const r=devicePixelRatio||1,w=c.clientWidth,h=c.clientHeight;c.width=w*r;c.height=h*r;const x=c.getContext('2d');x.setTransform(r,0,0,r,0,0);x.clearRect(0,0,w,h);return[x,w,h]}
function axes(x,w,h,P,xr,yr,fxx,fyy){x.font='11px system-ui';x.fillStyle=css('--mute');x.strokeStyle=css('--grid');x.lineWidth=1;
 const nt=(a,b,n)=>{const s=Math.pow(10,Math.floor(Math.log10((b-a)/n||1)));const m=[1,2,5,10].find(m=>(b-a)/(s*m)<=n)*s;const o=[];for(let v=Math.ceil(a/m)*m;v<=b+1e-9;v+=m)o.push(+v.toFixed(6));return o};
 nt(xr[0],xr[1],8).forEach(v=>{x.beginPath();x.moveTo(fxx(v),P.t);x.lineTo(fxx(v),h-P.b);x.stroke();x.fillText(v,fxx(v)-8,h-P.b+14)});
 nt(yr[0],yr[1],5).forEach(v=>{x.beginPath();x.moveTo(P.l,fyy(v));x.lineTo(w-P.r,fyy(v));x.stroke();x.fillText(v,4,fyy(v)+4)})}
function wiPoints(){if(!WI||!WI.path)return null;const T=J.takeoff,D=J.axis===0?[J.sign,0]:[0,J.sign],o=[];for(let i=0;i<WI.path.length;i+=4){const dx=WI.path[i]-T[0],dy=WI.path[i+1]-T[1];o.push({f:i/4,fwd:dx*D[0]+dy*D[1],side:-dx*D[1]+dy*D[0],h:WI.path[i+2]-T[2],spd:WI.path[i+3]})}return o}
function line(x,pts,kx,ky,fxx,fyy,col,w){x.strokeStyle=col;x.lineWidth=w;x.beginPath();pts.forEach((p,i)=>{const X=fxx(p[kx]),Y=fyy(p[ky]);i?x.lineTo(X,Y):x.moveTo(X,Y)});x.stroke()}
function drawCharts(){if(!J)return;const ch=J.chart,air=ch.filter(p=>p.air),wp=wiPoints();
 const space=(c,ky,yl)=>{const[x,w,h]=setup(c),P={l:40,r:12,t:10,b:24};const all=air.concat(wp||[]);
  let x0=Math.min(-20,...all.map(p=>p.fwd)),x1=Math.max(...all.map(p=>p.fwd))+10;let y0=Math.min(...all.map(p=>p[ky]))-5,y1=Math.max(...all.map(p=>p[ky]))+5;
  if(ky==='side'){const m=Math.max(Math.abs(y0),Math.abs(y1),20);y0=-m;y1=m}
  const fxx=v=>P.l+(v-x0)/(x1-x0)*(w-P.l-P.r),fyy=v=>h-P.b-(v-y0)/(y1-y0)*(h-P.t-P.b);axes(x,w,h,P,[x0,x1],[y0,y1],fxx,fyy);
  x.setLineDash([4,4]);x.strokeStyle=css('--mute');[J.edgeFwd,J.faceFwd].forEach(v=>{if(v!=null){x.beginPath();x.moveTo(fxx(v),P.t);x.lineTo(fxx(v),h-P.b);x.stroke()}});x.setLineDash([]);
  line(x,air,'fwd',ky,fxx,fyy,css('--air'),2);if(wp)line(x,wp,'fwd',ky,fxx,fyy,css('--wi'),2)};
 space($('c1'),'side');space($('c2'),'h');
 const time=(c,ky,sym)=>{const[x,w,h]=setup(c),P={l:44,r:12,t:10,b:24};const all=ky==='spd'?ch.concat(wp||[]):ch;
  const x0=Math.min(...ch.map(p=>p.f)),x1=Math.max(...ch.map(p=>p.f));let y0=Math.min(...all.map(p=>p[ky])),y1=Math.max(...all.map(p=>p[ky]));
  if(sym){const m=Math.max(Math.abs(y0),Math.abs(y1),1);y0=-m;y1=m}else{y0=Math.max(0,y0-10);y1+=10}
  const fxx=v=>P.l+(v-x0)/(x1-x0)*(w-P.l-P.r),fyy=v=>h-P.b-(v-y0)/(y1-y0)*(h-P.t-P.b);axes(x,w,h,P,[x0,x1],[y0,y1],fxx,fyy);
  line(x,ch,'f',ky,fxx,fyy,css('--ground'),1.5);line(x,air,'f',ky,fxx,fyy,css('--air'),2);if(wp&&ky==='spd')line(x,wp,'f',ky,fxx,fyy,css('--wi'),2)};
 time($('c3'),'spd');time($('c4'),'dy',true);
}
addEventListener('resize',()=>{drawCharts();TL.cache=null});

// ---------- 3D replay (renderer from LJSlapTool) ----------
const RC=$('rc'),HUD=$('rh'),msg=$('rmsg');const gl=RC.getContext('webgl2',{antialias:true});
let entKey='',fov=100,t=0,speed=0.25,playing=false,last=0,scene=null,bsp=null,mapName=null,rp=null,other=null;
__RENDERER__
let gprog,gbuf,gVP,gOff,gScl,gCol;
function initGhost(){const vs=`#version 300 es
in vec3 p;uniform mat4 vp;uniform vec3 off;uniform vec3 scl;void main(){gl_Position=vp*vec4(p*scl+off,1.0);}`;
 const fs=`#version 300 es
precision highp float;uniform vec4 col;out vec4 o;void main(){o=col;}`;
 const sh=(ty,s)=>{const x=gl.createShader(ty);gl.shaderSource(x,s);gl.compileShader(x);return x};
 gprog=gl.createProgram();gl.attachShader(gprog,sh(gl.VERTEX_SHADER,vs));gl.attachShader(gprog,sh(gl.FRAGMENT_SHADER,fs));gl.linkProgram(gprog);
 const P=[[-16,-16,-1],[16,-16,-1],[16,16,-1],[-16,16,-1],[-16,-16,1],[16,-16,1],[16,16,1],[-16,16,1]],F=[[0,1,2,3],[4,5,6,7],[0,1,5,4],[1,2,6,5],[2,3,7,6],[3,0,4,7]],v=[];
 for(const q of F)for(const k of [0,1,2,0,2,3])v.push(...P[q[k]]);
 gbuf=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,gbuf);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array(v),gl.STATIC_DRAW);
 gVP=gl.getUniformLocation(gprog,'vp');gOff=gl.getUniformLocation(gprog,'off');gScl=gl.getUniformLocation(gprog,'scl');gCol=gl.getUniformLocation(gprog,'col')}
function sampleR(r,tf){const n=r.d.length/9,i=Math.max(0,Math.min(n-2,Math.floor(tf))),u=Math.max(0,Math.min(1,tf-i)),a=i*9,b=(i+1)*9,d=r.d;
 const lerp=(x,y)=>x+(y-x)*u,ang=(x,y)=>{let q=y-x;while(q>180)q-=360;while(q<-180)q+=360;return x+q*u};
 return {eye:[lerp(d[a],d[b]),lerp(d[a+1],d[b+1]),lerp(d[a+2],d[b+2])],p:ang(d[a+3],d[b+3]),y:ang(d[a+4],d[b+4]),btn:d[a+6],spd:lerp(d[a+7],d[b+7]),og:d[a+8],i}}
function setReplay(keep){
 const wi=$('rmode').value==='wi'&&WI;const off=rp?t-rp.t0:-100;
 rp=wi?{d:WI.replay.d,t0:WI.replay.t0,t1:WI.replay.t1,map:WI.replay.map,ents:WI.replay.ents}:J.replay;
 other=wi?J.replay:(WI?{d:WI.replay.d,t0:WI.replay.t0,t1:WI.replay.t1}:null);
 $('rs').max=(rp.d.length/9-1)*10;t=keep?Math.max(0,Math.min(rp.d.length/9-1,rp.t0+off)):Math.max(0,rp.t0-100);playing=false;$('rp').textContent='Play';
 const mv=J.replay.moves||null,ek=JSON.stringify(rp.ents||{})+'|'+Object.keys(mv||{}).join(',');
 const setE=()=>{window.ENT_OVR=rp.ents||null;window.ENT_MOV=mv;entKey=ek};
 if(rp.map!==mapName||!bsp){setE();loadMap(rp.map)}
 else if(ek!==entKey){setE();scene=buildScene(bsp);upload(scene)}}
// offset of a moving brush model at replay time tf (frames of the real replay)
function movOff(m,tf){const M=J&&J.replay.moves&&J.replay.moves[m];if(!M)return[0,0,0];const n=M.length/3;
 const x=Math.max(0,Math.min(n-1,tf)),i=Math.min(n-2,Math.floor(x)),u=Math.max(0,Math.min(1,x-i));if(n<2)return[M[0],M[1],M[2]];
 return [0,1,2].map(c=>M[i*3+c]+(M[i*3+3+c]-M[i*3+c])*u)}
function render(){
 const dpr=devicePixelRatio||1,w=RC.clientWidth,h=RC.clientHeight;
 if(RC.width!==Math.round(w*dpr)||RC.height!==Math.round(h*dpr)){RC.width=Math.round(w*dpr);RC.height=Math.round(h*dpr);HUD.width=RC.width;HUD.height=RC.height}
 if(!gl)return;gl.viewport(0,0,RC.width,RC.height);gl.clearColor(0.32,0.42,0.6,1);gl.clear(gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT);
 const hx=HUD.getContext('2d');hx.setTransform(dpr,0,0,dpr,0,0);hx.clearRect(0,0,w,h);
 if(!rp||!scene)return;
 const s=sampleR(rp,t),cam=$('rcam').value;let vp;
 const dk0=(s.btn&4)?12:17,ctr=[s.eye[0],s.eye[1],s.eye[2]-dk0];
 if(cam==='fp')vp=viewProj(s.eye,s.p,s.y,0,w/h);
 else{const jy=J&&J.dir?Math.atan2(J.dir[1],J.dir[0])*180/Math.PI:s.y;const[cp,cy,cd]=cam==='chase'?[18,jy,150]:cam==='side'?[8,jy+90,230]:[89,jy,320];
  const f=dirs(cp,cy,0).f;vp=viewProj([ctr[0]-f[0]*cd,ctr[1]-f[1]*cd,ctr[2]-f[2]*cd],cp,cy,0,w/h)}
 gl.enable(gl.DEPTH_TEST);gl.enable(gl.BLEND);gl.blendFunc(gl.SRC_ALPHA,gl.ONE_MINUS_SRC_ALPHA);gl.useProgram(prog);gl.uniformMatrix4fv(uVP,false,vp);
 gl.activeTexture(gl.TEXTURE1);gl.bindTexture(gl.TEXTURE_2D,lmTex);gl.uniform1i(uLm,1);gl.uniform1i(uTex,0);
 const opaque=glb.filter(b=>b.alpha>=1&&!b.water),rest=glb.filter(b=>b.alpha<1||b.water);
 const tReal=J?J.replay.t0+(t-rp.t0):t;
 for(const b of opaque.concat(rest)){const mo=b.mov?movOff(b.mov,tReal):[0,0,0];gl.uniform3f(uMoff,mo[0],mo[1],mo[2]);gl.activeTexture(gl.TEXTURE0);gl.bindTexture(gl.TEXTURE_2D,b.t);gl.bindBuffer(gl.ARRAY_BUFFER,b.buf);
  gl.enableVertexAttribArray(aPos);gl.vertexAttribPointer(aPos,3,gl.FLOAT,false,28,0);gl.enableVertexAttribArray(aUv);gl.vertexAttribPointer(aUv,2,gl.FLOAT,false,28,12);gl.enableVertexAttribArray(aLm);gl.vertexAttribPointer(aLm,2,gl.FLOAT,false,28,20);
  gl.uniform1f(uAlpha,b.alpha);gl.uniform1f(uWater,b.water?1:0);gl.uniform1f(uCut,b.cut);gl.depthMask(b.alpha>=1&&!b.water);gl.drawArrays(gl.TRIANGLES,0,b.n)}
 gl.depthMask(true);
 if(other&&$('gh').checked){const so=sampleR(other,other.t0+(t-rp.t0));const dk=(so.btn&4)?12:17;
  gl.useProgram(gprog);gl.disableVertexAttribArray(aUv);gl.disableVertexAttribArray(aLm);gl.bindBuffer(gl.ARRAY_BUFFER,gbuf);gl.enableVertexAttribArray(0);gl.vertexAttribPointer(0,3,gl.FLOAT,false,12,0);
  gl.uniformMatrix4fv(gVP,false,vp);gl.uniform3f(gOff,so.eye[0],so.eye[1],so.eye[2]-dk);gl.uniform3f(gScl,1,1,dk===12?18:36);
  const wiMode=$('rmode').value==='wi';gl.uniform4f(gCol,...(wiMode?[0.95,0.6,0.2,0.4]:[0.6,0.4,1.0,0.45]));gl.depthMask(false);gl.drawArrays(gl.TRIANGLES,0,36);gl.depthMask(true)}
 if(cam!=='fp'){gl.useProgram(gprog);gl.disableVertexAttribArray(aUv);gl.disableVertexAttribArray(aLm);gl.bindBuffer(gl.ARRAY_BUFFER,gbuf);gl.enableVertexAttribArray(0);gl.vertexAttribPointer(0,3,gl.FLOAT,false,12,0);
  gl.uniformMatrix4fv(gVP,false,vp);gl.uniform3f(gOff,ctr[0],ctr[1],ctr[2]);gl.uniform3f(gScl,1,1,dk0===12?18:36);
  gl.uniform4f(gCol,...(s.og?[0.15,1.0,0.55,0.55]:[1.0,0.6,0.2,0.55]));gl.depthMask(false);gl.drawArrays(gl.TRIANGLES,0,36);gl.depthMask(true)}
 hx.font='600 34px system-ui';hx.textAlign='center';hx.fillStyle='#fff';hx.shadowColor='#000';hx.shadowBlur=6;hx.fillText(Math.round(s.spd),w/2,h-46);hx.font='12px system-ui';hx.fillText('units/s',w/2,h-28);
 const B=s.btn,keys=[['W',8,1,0],['A',512,0,1],['S',16,1,1],['D',1024,2,1]],bx=24,by=h-92;hx.shadowBlur=0;hx.font='600 13px system-ui';
 const key=(lab,on,x,y,wd)=>{hx.fillStyle=on?'rgba(255,169,77,.95)':'rgba(0,0,0,.45)';hx.fillRect(x,y,wd,30);hx.fillStyle=on?'#000':'#fff';hx.fillText(lab,x+wd/2,y+20)};
 keys.forEach(k=>key(k[0],B&k[1],bx+k[2]*34,by+k[3]*34,30));key('JUMP',B&2,bx+110,by,60);key('DUCK',B&4,bx+110,by+34,60);
 const i=s.i,wiMode=$('rmode').value==='wi';hx.textAlign='right';hx.fillStyle='#fff';hx.shadowColor='#000';hx.shadowBlur=4;hx.font='12px system-ui';
 hx.fillText((i<rp.t0?'ground':i>rp.t1?'end':'air')+'  frame '+(Math.floor(t)-rp.t0),w-14,h-16);
 if(wiMode){hx.textAlign='center';hx.font='700 13px system-ui';hx.shadowBlur=0;hx.fillStyle='rgba(0,0,0,.5)';hx.fillRect(w/2-150,10,300,24);hx.fillStyle='#b794ff';hx.fillText('WHAT-IF · SIMULATED FROM TAKE-OFF',w/2,27)}
 if(i>rp.t1-1){const L=wiMode?[(WI.known?(WI.landed?'LANDS':'FAILS'):'?')+' (simulated)','Distance '+WI.dist.toFixed(3),'Maxspeed '+WI.max.toFixed(1),'Prestrafe '+WI.pre.toFixed(2),'Jumpoff '+(WI.edge==null?'—':WI.edge.toFixed(2))]
   :[(J.landed?'':'FAILED ')+J.type,'Distance '+J.dist.toFixed(3),'Maxspeed '+J.max.toFixed(1)+' (+'+J.gainTotal.toFixed(1)+')','Prestrafe '+J.pre.toFixed(3),'Strafes '+J.strafes.length+'  Eff '+J.eff.toFixed(0)+'%','Jumpoff '+fx(J.jumpoff)];
  hx.shadowBlur=0;hx.fillStyle='rgba(0,0,0,.55)';hx.fillRect(w-250,14,236,L.length*20+14);hx.textAlign='left';hx.font='13px ui-monospace,monospace';
  L.forEach((q,k)=>{hx.fillStyle=k===0?(wiMode?'#b794ff':(J.landed?'rgb(20,255,150)':'rgb(255,70,120)')):'rgb(20,255,150)';hx.fillText(q,w-240,36+k*20)})}
 $('rs').value=Math.round(t*10);drawTL()}
function loop(now){if(playing&&rp){const dt=Math.min(0.05,(now-last)/1000);t+=dt*100*speed;const n=rp.d.length/9;if(t>=n-1){t=n-1;playing=false;$('rp').textContent='Play'}}last=now;render();requestAnimationFrame(loop)}
async function loadMap(name){mapName=name;msg.textContent='Loading map '+name+' ...';msg.style.display='block';
 try{const r=await fetch('/map/'+encodeURIComponent(name)+'.bsp');if(!r.ok)throw new Error('not found');useBuf(await r.arrayBuffer())}
 catch(e){msg.innerHTML='Map <b>'+name+'.bsp</b> was not found. Put it next to DEMviewer.exe or the demos, or pick it here: <input type="file" id="rf" accept=".bsp">';
  $('rf').onchange=async ev=>{const f=ev.target.files[0];if(f)useBuf(await f.arrayBuffer())}}}
function useBuf(buf){try{bsp=parseBSP(buf);scene=buildScene(bsp);upload(scene);msg.style.display='none'}catch(e){msg.textContent='Could not read map: '+e.message;msg.style.display='block'}}
if(!gl){msg.textContent='3D view needs WebGL2.';msg.style.display='block'}else{try{initGL();initGhost()}catch(e){msg.textContent='3D view not available: '+e.message;msg.style.display='block'}}
$('rp').onclick=()=>{if(!rp)return;if(playing){playing=false;$('rp').textContent='Play';return}if(t>=rp.d.length/9-2)t=Math.max(0,rp.t0-100);playing=true;$('rp').textContent='Pause';last=performance.now()};
$('rs').oninput=e=>{t=e.target.value/10;playing=false;$('rp').textContent='Play'};
$('rv').onchange=e=>{speed=+e.target.value};$('rcam').onchange=()=>{};$('rf2').oninput=e=>{fov=+e.target.value;$('rfl').textContent=fov};
$('rt').onclick=()=>{if(rp){t=Math.max(0,rp.t0-30);playing=true;$('rp').textContent='Pause';last=performance.now()}};
requestAnimationFrame(loop);loadState();

// ---- timeline: yaw speed + keys, one block per frame ----
const TL={a:0,b:0,cache:null,key:'',drag:null};
function tlReset(){if(!J)return;const n=J.replay.d.length/9;TL.a=Math.max(0,J.replay.t0-15);TL.b=Math.min(n,J.replay.t1+15);TL.cache=null}
function tlLanes(){const R=J.replay,d=R.d,n=d.length/9,B=k=>d[k*9+6];
 const lane=(name,fn,col)=>({name,on:k=>fn(k),col});
 return [lane('use',k=>B(k)&32),
  {name:'techniques',tech:true},
  lane('jump',k=>B(k)&2),lane('ground',k=>d[k*9+8]>0),lane('duck',k=>B(k)&4),
  lane('duckstate',k=>R.duck&&R.duck[k],'#22e39a'),
  lane('forward',k=>B(k)&8),lane('back',k=>B(k)&16),lane('moveleft',k=>B(k)&512),lane('moveright',k=>B(k)&1024),lane('dead air',k=>k>R.t0&&k<R.t1&&!(B(k)&(8|16|512|1024)),'#ff5c5c')]}
function drawTL(){const cv=$('tl');if(!cv||!J||$('detail').style.display==='none')return;
 const dpr=devicePixelRatio||1,W=cv.clientWidth,H=cv.clientHeight;if(!W)return;
 if(cv.width!==Math.round(W*dpr)||cv.height!==Math.round(H*dpr)){cv.width=Math.round(W*dpr);cv.height=Math.round(H*dpr);TL.cache=null}
 const ck=J._d+'_'+J._j+'_'+TL.a.toFixed(2)+'_'+TL.b.toFixed(2)+'_'+W+'_'+H;
 if(TL.cache&&TL.key===ck){const x=cv.getContext('2d');x.setTransform(1,0,0,1,0,0);x.drawImage(TL.cache,0,0);tlCursor(x,dpr,W,H);return}
 const off=document.createElement('canvas');off.width=cv.width;off.height=cv.height;const g=off.getContext('2d');g.setTransform(dpr,0,0,dpr,0,0);
 const R=J.replay,d=R.d,n=d.length/9,L=118,Rm=12,top=8,gh=170,laneH=24,lanes=tlLanes(),ay0=top+gh+16,ah=110,ly0=ay0+ah+14,axY=ly0+lanes.length*laneH+4;
 const a=TL.a,b=TL.b,fx=f=>L+(f-a)/(b-a)*(W-L-Rm),fw=(W-L-Rm)/(b-a);
 const ink=css('--ink'),mute=css('--mute'),line=css('--line'),grid=css('--grid'),air=css('--air');
 const LB=[];const lab=(x,y,t,al,c)=>LB.push(`<span class="tl-l ${c||''}" style="left:${x}px;top:${y}px;transform:translate(${al==='r'?'-100%':al==='c'?'-50%':'0'},-50%)">${t}</span>`);
 // yaw speed (deg per frame, + = turning left)
 const yaw=k=>{if(k<1)return 0;let q=d[k*9+4]-d[(k-1)*9+4];while(q>180)q-=360;while(q<-180)q+=360;return q};
 let mx=2;for(let k=Math.max(1,Math.floor(a));k<Math.min(n,Math.ceil(b)+1);k++)mx=Math.max(mx,Math.abs(yaw(k)));mx=Math.ceil(mx);
 const gy=v=>top+gh/2-v/mx*(gh/2);
 for(let v=-mx;v<=mx;v+=Math.max(1,Math.round(mx/3))){g.strokeStyle=grid;g.beginPath();g.moveTo(L,gy(v));g.lineTo(W-Rm,gy(v));g.stroke();lab(L-8,gy(v),v,'r','num')}
 LB.push(`<span class="tl-l tl-rot" style="left:14px;top:${top+gh/2}px">yaw speed (°/frame)</span>`);
 g.strokeStyle=line;g.beginPath();g.moveTo(L,gy(0));g.lineTo(W-Rm,gy(0));g.stroke();
 g.save();g.beginPath();g.rect(L,top,W-L-Rm,gh);g.clip();g.strokeStyle=air;g.lineWidth=1.6;g.beginPath();
 for(let k=Math.max(1,Math.floor(a)-1);k<Math.min(n,Math.ceil(b)+2);k++){const X=fx(k+0.5),Y=gy(yaw(k));k===Math.max(1,Math.floor(a)-1)?g.moveTo(X,Y):g.lineTo(X,Y)}g.stroke();g.restore();
 // yaw acceleration (change of yaw speed per frame, °/frame²)
 const yac=k=>k<2?0:yaw(k)-yaw(k-1);
 let ma=1;for(let k=Math.max(2,Math.floor(a));k<Math.min(n,Math.ceil(b)+1);k++)ma=Math.max(ma,Math.abs(yac(k)));ma=Math.ceil(ma);
 const ay=v=>ay0+ah/2-v/ma*(ah/2);
 for(let v=-ma;v<=ma;v+=Math.max(1,Math.round(ma/2))){g.strokeStyle=grid;g.beginPath();g.moveTo(L,ay(v));g.lineTo(W-Rm,ay(v));g.stroke();lab(L-8,ay(v),v,'r','num')}
 LB.push(`<span class="tl-l tl-rot" style="left:14px;top:${ay0+ah/2}px">yaw accel (°/frame²)</span>`);
 g.strokeStyle=line;g.beginPath();g.moveTo(L,ay(0));g.lineTo(W-Rm,ay(0));g.stroke();
 g.save();g.beginPath();g.rect(L,ay0,W-L-Rm,ah);g.clip();g.strokeStyle='#22c3d6';g.lineWidth=1.6;g.beginPath();
 for(let k=Math.max(2,Math.floor(a)-1);k<Math.min(n,Math.ceil(b)+2);k++){const X=fx(k+0.5),Y=ay(yac(k));k===Math.max(2,Math.floor(a)-1)?g.moveTo(X,Y):g.lineTo(X,Y)}g.stroke();g.restore();
 // lanes
 const gap=fw>=2.5?1:0;
 lanes.forEach((ln,i)=>{const y=ly0+i*laneH;lab(L-10,y+laneH/2-1,ln.name,'r');
  g.strokeStyle=line;g.strokeRect(L+0.5,y+0.5,W-L-Rm-1,laneH-3);
  if(ln.tech){const x0=Math.max(L,fx(R.t0)),x1=Math.min(W-Rm,fx(R.t1+1));if(x1<=x0)return;g.fillStyle=J.landed?'#2e9b3a':'#b8323f';g.fillRect(x0,y+2,x1-x0,laneH-7);
   lab((x0+x1)/2,y+laneH/2-1,J.type.toLowerCase(),'c','onbar');return}
  g.fillStyle=ln.col||'#6b7280';
  for(let k=Math.max(0,Math.floor(a));k<Math.min(n,Math.ceil(b));k++){if(!ln.on(k))continue;const X0=Math.round(Math.max(L,fx(k))),X1=Math.round(Math.min(W-Rm,fx(k+1)));if(X1<=X0)continue;g.fillRect(X0+gap,y+2,Math.max(1,X1-X0-gap),laneH-7)}
  if(ln.name==='jump'&&R.t0>=a&&R.t0<=b){g.fillStyle='#39ff5a';g.fillRect(fx(R.t0),y+1,2,laneH-5)}
  if(ln.name==='duck'){for(let k=Math.max(1,Math.ceil(a));k<Math.min(n,Math.floor(b));k++){const p=d[(k-1)*9+6]&4,c=d[k*9+6]&4;
   if(c&&!p){g.fillStyle='#ff3b3b';g.fillRect(fx(k),y+1,2,laneH-5)}if(p&&!c){g.fillStyle='#3b6bff';g.fillRect(fx(k),y+1,2,laneH-5)}}}});
 // frame axis, numbered from the take-off
 const span=b-a,stp=[1,2,5,10,20,25,50,100].find(s=>span/s<=24)||200;
 for(let f=Math.ceil((a-R.t0)/stp)*stp;f+R.t0<=b;f+=stp){const X=fx(f+R.t0+0.5);lab(X,axY+10,f,'c','num');g.strokeStyle=grid;g.beginPath();g.moveTo(X,axY);g.lineTo(X,axY+3);g.stroke()}
 lab(L,axY+28,'frames from take-off','l');
 TL.cache=off;TL.key=ck;$('tllab').innerHTML=LB.join('');TL.geo={L,Rm,top,axY,a,b,W,ly0,laneH,lanes:lanes.map(l=>l.name)};
 const x=cv.getContext('2d');x.setTransform(1,0,0,1,0,0);x.clearRect(0,0,cv.width,cv.height);x.drawImage(off,0,0);tlCursor(x,dpr,W,H)}
function tlCursor(x,dpr,W,H){if(!rp||!TL.geo)return;const G=TL.geo,tr=J.replay.t0+(t-rp.t0);if(tr<G.a||tr>G.b)return;
 const X=G.L+(tr-G.a)/(G.b-G.a)*(W-G.L-G.Rm);x.save();x.setTransform(dpr,0,0,dpr,0,0);x.strokeStyle=css('--ink');x.setLineDash([4,4]);x.beginPath();x.moveTo(X,G.top);x.lineTo(X,G.axY);x.stroke();x.restore()}
(()=>{const cv=$('tl');if(!cv)return;
 const fAt=e=>{const r=cv.getBoundingClientRect(),G=TL.geo;if(!G)return 0;return G.a+(e.clientX-r.left-G.L)/(r.width-G.L-G.Rm)*(G.b-G.a)};
 cv.addEventListener('wheel',e=>{if(!J)return;e.preventDefault();const n=J.replay.d.length/9,f=fAt(e),z=e.deltaY>0?1.25:0.8;
  let a=f-(f-TL.a)*z,b=f+(TL.b-f)*z;if(b-a<12){const m=(a+b)/2;a=m-6;b=m+6}if(a<0){b-=a;a=0}if(b>n){a-=b-n;b=n}TL.a=Math.max(0,a);TL.b=Math.min(n,b);drawTL()},{passive:false});
 cv.addEventListener('mousedown',e=>{TL.drag={x:e.clientX,a:TL.a,b:TL.b,moved:false}});
 addEventListener('mousemove',e=>{const D=TL.drag;if(!D||!J)return;const dx=e.clientX-D.x;if(Math.abs(dx)>3)D.moved=true;if(!D.moved)return;
  const r=cv.getBoundingClientRect(),G=TL.geo,n=J.replay.d.length/9,df=-dx/(r.width-G.L-G.Rm)*(D.b-D.a);let a=D.a+df,b=D.b+df;
  if(a<0){b-=a;a=0}if(b>n){a-=b-n;b=n}TL.a=a;TL.b=b;drawTL()});
 addEventListener('mouseup',e=>{const D=TL.drag;TL.drag=null;if(!D||D.moved||!J||!rp)return;const f=fAt(e);
  t=Math.max(0,Math.min(rp.d.length/9-1,rp.t0+(f-J.replay.t0)));playing=false;$('rp').textContent='Play'});
})();

(()=>{const cv=$('tl');if(!cv)return;const tip=document.createElement('div');tip.id='tltip';document.body.appendChild(tip);
 const keys=k=>{const B=J.replay.d[k*9+6],o=[];[[8,'forward'],[16,'back'],[512,'moveleft'],[1024,'moveright'],[2,'jump'],[4,'duck'],[32,'use']].forEach(([m,n])=>{if(B&m)o.push(n)});return o.join(', ')||'none'};
 const row=(k,v)=>`<tr><td>${k}</td><td><b>${v}</b></td></tr>`;
 cv.addEventListener('mousemove',e=>{if(!J||!TL.geo||TL.drag&&TL.drag.moved){tip.style.display='none';return}
  const G=TL.geo,r=cv.getBoundingClientRect(),x=e.clientX-r.left,y=e.clientY-r.top;if(x<G.L||x>G.W-G.Rm){tip.style.display='none';return}
  const R=J.replay,f=Math.floor(G.a+(x-G.L)/(G.W-G.L-G.Rm)*(G.b-G.a)),n=R.d.length/9;if(f<0||f>=n){tip.style.display='none';return}
  const li=Math.floor((y-G.ly0)/G.laneH),lane=y>=G.ly0&&li>=0&&li<G.lanes.length?G.lanes[li]:null;let h;
  if(lane==='techniques'&&f>=R.t0&&f<=R.t1){const s=J.strafes||[];
   h=`<div class="tt-h" style="color:${J.landed?'var(--land)':'var(--bad)'}">${J.landed?'':'FAILED '}${J.type} ${fx(J.dist)}</div><table>`+
    row('Prestrafe',fx(J.pre))+row('Maxspeed',fx(J.max,1)+' (+'+fx(J.gainTotal,1)+')')+row('Strafes',s.length)+row('Eff',(J.eff||0).toFixed(0)+'%')+
    row('Jumpoff',fx(J.jumpoff))+row('Block',J.block==null?'—':J.block.toFixed(0))+row('Landing',fx(J.landEdge))+row('FOG',J.fog)+row('Air time',J.airFrames+' frames')+row('Dead airtime',(J.deadAir||0)+' frames')+
    (()=>{const ya=yawAccStats();return ya?row('Yaw accel (switch)',ya.avg.toFixed(2)+' avg / '+ya.max.toFixed(2)+' max'):''})()+row('Mouse slaps',J.oneFrameSlaps+' / '+Math.max(J.slaps-1,0))+'</table>'}
  else{let q=0;if(f>=1){q=R.d[f*9+4]-R.d[(f-1)*9+4];while(q>180)q-=360;while(q<-180)q+=360}
   h=`<div class="tt-h">frame ${f-R.t0>0?'+':''}${f-R.t0}</div><table>`+row('Yaw speed',q.toFixed(2)+'°')+row('Yaw accel',(f>=2?q-(()=>{let p=R.d[(f-1)*9+4]-R.d[(f-2)*9+4];while(p>180)p-=360;while(p<-180)p+=360;return p})():0).toFixed(2)+'°')+row('Speed',Math.round(R.d[f*9+7])+' u/s')+row('Keys',keys(f))+row('On ground',R.d[f*9+8]?'yes':'no')+'</table>'}
  tip.innerHTML=h;tip.style.display='block';const tw=tip.offsetWidth,th=tip.offsetHeight;
  tip.style.left=Math.min(innerWidth-tw-8,e.clientX+14)+'px';tip.style.top=Math.max(8,Math.min(innerHeight-th-8,e.clientY+14))+'px'});
 cv.addEventListener('mouseleave',()=>{tip.style.display='none'});
})();

function yawAccStats(){const R=J.replay,d=R.d,n=d.length/9,yaw=k=>{let q=d[k*9+4]-d[(k-1)*9+4];while(q>180)q-=360;while(q<-180)q+=360;return q};
 const pk=[];let prev=0;for(let k=Math.max(2,R.t0+1);k<Math.min(n,R.t1);k++){const v=yaw(k);if(Math.abs(v)<0.05)continue;const sg=Math.sign(v);
  if(prev&&sg!==prev){let m=0;for(let q=Math.max(2,k-2);q<=Math.min(n-1,k+2);q++)m=Math.max(m,Math.abs(yaw(q)-yaw(q-1)));pk.push(m)}prev=sg}
 if(!pk.length)return null;return {avg:pk.reduce((x,y)=>x+y,0)/pk.length,max:Math.max(...pk),n:pk.length}}

function cvarBoxes(){const C=J.cvars||[];if(!C.length)return'';const bad=C.filter(c=>c.ok===false),chk=C.filter(c=>c.ok!==null&&c.ok!==undefined),unk=C.filter(c=>c.ok===null||c.ok===undefined);
 const esc=s=>String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/"/g,'&quot;');
 const tip=C.map(c=>`${c.ok===false?'✗':c.ok?'✓':'?'} ${c.name}: ${c.value||'—'} (legal ${c.legal})${c.note?' · '+c.note:''}`).join('\n');
 let h=`<div class="stat" title="${esc(tip)}" style="cursor:help"><b style="color:${bad.length?'var(--bad)':'var(--land)'}">${bad.length?'✗ not legal':'✓ legal'}</b><span>settings · ${chk.length} checked${unk.length?', '+unk.length+' not in demo':''} · hover for list</span></div>`;
 if(bad.length)h+=`<div class="stat" style="border-color:var(--bad)"><b style="color:var(--bad);font-size:15px">${bad.map(c=>esc(c.name)).join(', ')}</b><span>${bad.map(c=>esc(c.name)+' '+esc(c.value)+' (legal '+esc(c.legal)+')').join(' · ')}</span></div>`;
 return h}
