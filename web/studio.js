// ---------- GoldSrc studio models (.mdl v10): parse, animate, draw ----------
// Models are read from the player's own cstrike/models folder through /mdl/.
const MDL={cache:new Map()};
function mdlParse(buf,texBuf){
 const dv=new DataView(buf),u8=new Uint8Array(buf),I=o=>dv.getInt32(o,true),F=o=>dv.getFloat32(o,true),S=o=>dv.getInt16(o,true),US=o=>dv.getUint16(o,true);
 const str=(o,n)=>{let s='';for(let i=0;i<n&&u8[o+i];i++)s+=String.fromCharCode(u8[o+i]);return s};
 if(str(0,4)!=='IDST'||I(4)!==10)throw new Error('not a GoldSrc model');
 const M={bones:[],seqs:[],tex:[],parts:[],skins:[],dv,u8};
 for(let k=0,o=I(144);k<I(140);k++,o+=112){const v=[],sc=[];for(let j=0;j<6;j++){v.push(F(o+64+j*4));sc.push(F(o+88+j*4))}
  M.bones.push({name:str(o,32),parent:I(o+32),value:v,scale:sc})}
 for(let k=0,o=I(168);k<I(164);k++,o+=176)M.seqs.push({name:str(o,32).toLowerCase(),fps:F(o+32),flags:I(o+36),frames:I(o+56),lin:[F(o+76),F(o+80),F(o+84)],blends:I(o+120),anim:I(o+124),group:I(o+156)});
 // textures: in this file, or in the <name>T.mdl companion
 const td=texBuf?new DataView(texBuf):dv,tu=texBuf?new Uint8Array(texBuf):u8,TI=o=>td.getInt32(o,true);
 for(let k=0,o=TI(184);k<TI(180);k++,o+=80){const w=TI(o+68),h=TI(o+72),fl=TI(o+64),di=TI(o+76),pal=di+w*h,rgba=new Uint8Array(w*h*4);
  for(let i=0;i<w*h;i++){const c=tu[di+i];rgba[i*4]=tu[pal+c*3];rgba[i*4+1]=tu[pal+c*3+1];rgba[i*4+2]=tu[pal+c*3+2];rgba[i*4+3]=(fl&0x40)&&c===255?0:255}
  M.tex.push({w,h,rgba,masked:!!(fl&0x40)})}
 const nref=TI(192);for(let k=0;k<nref;k++)M.skins.push(td.getInt16(TI(200)+k*2,true));
 // body parts: first sub-model of each (default body)
 for(let k=0,o=I(208);k<I(204);k++,o+=76){const mi=I(o+72);const mo=mi;
  const nverts=I(mo+80),vinfo=I(mo+84),vidx=I(mo+88),nn=I(mo+92),ninfo=I(mo+96),nidx=I(mo+100);
  const vb=new Uint8Array(nverts),vp=new Float32Array(nverts*3),nb=new Uint8Array(nn),np=new Float32Array(nn*3);
  for(let i=0;i<nverts;i++){vb[i]=u8[vinfo+i];vp[i*3]=F(vidx+i*12);vp[i*3+1]=F(vidx+i*12+4);vp[i*3+2]=F(vidx+i*12+8)}
  for(let i=0;i<nn;i++){nb[i]=u8[ninfo+i];np[i*3]=F(nidx+i*12);np[i*3+1]=F(nidx+i*12+4);np[i*3+2]=F(nidx+i*12+8)}
  const meshes=[];
  for(let m=0,mo2=I(mo+76);m<I(mo+72);m++,mo2+=20){const skin=I(mo2+8),tex=M.skins[skin]!=null?M.skins[skin]:skin,tw=(M.tex[tex]||{w:1}).w,th=(M.tex[tex]||{h:1}).h;
   const tris=[];let p=I(mo2+4);
   for(;;){let n=S(p);p+=2;if(!n)break;const fan=n<0;n=Math.abs(n);const vs=[];
    for(let i=0;i<n;i++){vs.push([US(p),US(p+2),S(p+4)/tw,S(p+6)/th]);p+=8}
    for(let i=2;i<n;i++){if(fan)tris.push(vs[0],vs[i-1],vs[i]);else if(i&1)tris.push(vs[i-1],vs[i-2],vs[i]);else tris.push(vs[i-2],vs[i-1],vs[i])}}
   meshes.push({tex,tris})}
  M.parts.push({vb,vp,nb,np,meshes})}
 return M}
// one animation value of bone b, channel j (0-2 pos, 3-5 rot) at integer frame f
function mdlAnimVal(M,animOff,b,j,f){const bone=M.bones[b],dv=M.dv,base=animOff+b*12,off=dv.getUint16(base+j*2,true);
 if(!off)return bone.value[j];let p=base+off,k=f;
 for(let guard=0;guard<1000;guard++){const valid=M.u8[p],total=M.u8[p+1];if(total>k)break;k-=total;p+=(valid+1)*2;if(!total)break}
 const valid=M.u8[p],v=valid>k?dv.getInt16(p+(k+1)*2,true):dv.getInt16(p+valid*2,true);return bone.value[j]+v*bone.scale[j]}
function eulerQ(x,y,z){const sr=Math.sin(x/2),cr=Math.cos(x/2),sp=Math.sin(y/2),cp=Math.cos(y/2),sy=Math.sin(z/2),cy=Math.cos(z/2);
 return[sr*cp*cy-cr*sp*sy,cr*sp*cy+sr*cp*sy,cr*cp*sy-sr*sp*cy,cr*cp*cy+sr*sp*sy]}
function qslerp(a,b,t){let d=a[0]*b[0]+a[1]*b[1]+a[2]*b[2]+a[3]*b[3];if(d<0){b=b.map(x=>-x);d=-d}
 if(d>0.9995){const q=a.map((x,i)=>x+(b[i]-x)*t),l=Math.hypot(...q);return q.map(x=>x/l)}
 const th=Math.acos(d),s=Math.sin(th),wa=Math.sin((1-t)*th)/s,wb=Math.sin(t*th)/s;return a.map((x,i)=>x*wa+b[i]*wb)}
// local pose (pos + quat per bone) of a sequence at a fractional frame and blend
function mdlPose(M,si,frame,blend){const s=M.seqs[si];if(!s)return null;const nb=M.bones.length,fr=Math.max(0,Math.min(s.frames-1,frame)),f0=Math.floor(fr),f1=Math.min(s.frames-1,f0+1),t=fr-f0;
 const one=bi=>{const a=s.anim+bi*nb*12,P=[],Q=[];for(let b=0;b<nb;b++){const pa=[0,1,2].map(j=>mdlAnimVal(M,a,b,j,f0)),pb=[0,1,2].map(j=>mdlAnimVal(M,a,b,j,f1));
   const qa=eulerQ(...[3,4,5].map(j=>mdlAnimVal(M,a,b,j,f0))),qb=eulerQ(...[3,4,5].map(j=>mdlAnimVal(M,a,b,j,f1)));
   P.push(pa.map((x,i)=>x+(pb[i]-x)*t));Q.push(qslerp(qa,qb,t))}return{P,Q}};
 if(s.blends<=1)return one(0);
 const x=Math.max(0,Math.min(s.blends-1,blend*(s.blends-1))),b0=Math.floor(x),b1=Math.min(s.blends-1,b0+1),w=x-b0,A=one(b0);if(w<0.01)return A;const B=one(b1);
 return{P:A.P.map((p,i)=>p.map((v,j)=>v+(B.P[i][j]-v)*w)),Q:A.Q.map((q,i)=>qslerp(q,B.Q[i],w))}}
// bone matrices (3x4, row major) from a pose, optionally taking some bones from another skeleton by name
function mdlBones(M,pose,merge){const nb=M.bones.length,out=new Array(nb);
 for(let b=0;b<nb;b++){if(merge&&merge.has(M.bones[b].name)){out[b]=merge.get(M.bones[b].name);continue}
  const[x,y,z,w]=pose.Q[b],p=pose.P[b];
  const m=[1-2*(y*y+z*z),2*(x*y-w*z),2*(x*z+w*y),p[0], 2*(x*y+w*z),1-2*(x*x+z*z),2*(y*z-w*x),p[1], 2*(x*z-w*y),2*(y*z+w*x),1-2*(x*x+y*y),p[2]];
  const par=M.bones[b].parent;if(par<0){out[b]=m;continue}const a=out[par],r=new Array(12);
  for(let i=0;i<3;i++){for(let j=0;j<3;j++)r[i*4+j]=a[i*4]*m[j]+a[i*4+1]*m[4+j]+a[i*4+2]*m[8+j];r[i*4+3]=a[i*4]*m[3]+a[i*4+1]*m[7]+a[i*4+2]*m[11]+a[i*4+3]}
  out[b]=r}return out}
function mdlSeq(M,name){return M.seqs.findIndex(s=>s.name===name)}
// ---- GL side ----
let sprog,sPos,sNor,sUv,sVP,sTex,sTint,sCut;
function initStudioGL(){const vs=`#version 300 es
in vec3 p;in vec3 n;in vec2 uv;uniform mat4 vp;out vec2 vuv;out float vl;
void main(){vuv=uv;vec3 L=normalize(vec3(0.4,0.3,0.85));vl=0.55+0.5*max(dot(normalize(n),L),0.0);gl_Position=vp*vec4(p,1.0);}`;
 const fs=`#version 300 es
precision highp float;in vec2 vuv;in float vl;uniform sampler2D tex;uniform vec4 tint;uniform float cut;out vec4 o;
void main(){vec4 c=texture(tex,vuv);if(cut>0.5&&c.a<0.5)discard;o=vec4(mix(c.rgb*vl,tint.rgb,tint.a),1.0);}`;
 const sh=(t,s)=>{const x=gl.createShader(t);gl.shaderSource(x,s);gl.compileShader(x);if(!gl.getShaderParameter(x,gl.COMPILE_STATUS))throw new Error(gl.getShaderInfoLog(x));return x};
 sprog=gl.createProgram();gl.attachShader(sprog,sh(gl.VERTEX_SHADER,vs));gl.attachShader(sprog,sh(gl.FRAGMENT_SHADER,fs));gl.linkProgram(sprog);
 sPos=gl.getAttribLocation(sprog,'p');sNor=gl.getAttribLocation(sprog,'n');sUv=gl.getAttribLocation(sprog,'uv');
 sVP=gl.getUniformLocation(sprog,'vp');sTex=gl.getUniformLocation(sprog,'tex');sTint=gl.getUniformLocation(sprog,'tint');sCut=gl.getUniformLocation(sprog,'cut')}
// fetch + parse a model by path under cstrike/models ("player/gign/gign.mdl"); null while loading or when missing
function mdlGet(path){if(!path)return null;let e=MDL.cache.get(path);if(e)return e.M||null;e={M:null};MDL.cache.set(path,e);
 (async()=>{try{const r=await fetch('/mdl/'+path);if(!r.ok)return;const buf=await r.arrayBuffer();let tb=null;
   if(new DataView(buf).getInt32(180,true)===0){const r2=await fetch('/mdl/'+path.replace(/\.mdl$/i,'T.mdl'));if(r2.ok)tb=await r2.arrayBuffer()}
   const M=mdlParse(buf,tb);M.gltex=M.tex.map(t=>{const x=gl.createTexture();gl.bindTexture(gl.TEXTURE_2D,x);gl.texImage2D(gl.TEXTURE_2D,0,gl.RGBA,t.w,t.h,0,gl.RGBA,gl.UNSIGNED_BYTE,t.rgba);
    gl.generateMipmap(gl.TEXTURE_2D);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MIN_FILTER,gl.LINEAR_MIPMAP_LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MAG_FILTER,gl.LINEAR);return x});
   M.buf=gl.createBuffer();e.M=M}catch(err){console.log('model '+path+': '+err.message)}})();return null}
// draw a posed model; X maps model space to world: [ox,oy,oz, fx,fy,fz, lx,ly,lz, ux,uy,uz]
function mdlDraw(M,bones,X,vp,tint){const data=[],batches=[];
 for(const part of M.parts)for(const mesh of part.meshes){const start=data.length/8;
  for(const[vi,ni,s,t]of mesh.tris){const b=bones[part.vb[vi]]||bones[0],v=[part.vp[vi*3],part.vp[vi*3+1],part.vp[vi*3+2]],nb=bones[part.nb[ni]]||b,n=[part.np[ni*3],part.np[ni*3+1],part.np[ni*3+2]];
   const mx=b[0]*v[0]+b[1]*v[1]+b[2]*v[2]+b[3],my=b[4]*v[0]+b[5]*v[1]+b[6]*v[2]+b[7],mz=b[8]*v[0]+b[9]*v[1]+b[10]*v[2]+b[11];
   const nx=nb[0]*n[0]+nb[1]*n[1]+nb[2]*n[2],ny=nb[4]*n[0]+nb[5]*n[1]+nb[6]*n[2],nz=nb[8]*n[0]+nb[9]*n[1]+nb[10]*n[2];
   data.push(X[0]+mx*X[3]+my*X[6]+mz*X[9],X[1]+mx*X[4]+my*X[7]+mz*X[10],X[2]+mx*X[5]+my*X[8]+mz*X[11],
     nx*X[3]+ny*X[6]+nz*X[9],nx*X[4]+ny*X[7]+nz*X[10],nx*X[5]+ny*X[8]+nz*X[11],s,t)}
  batches.push({tex:mesh.tex,start,n:data.length/8-start})}
 gl.useProgram(sprog);gl.bindBuffer(gl.ARRAY_BUFFER,M.buf);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array(data),gl.STREAM_DRAW);
 for(const a of[0,1,2,3,4,5,6,7])gl.disableVertexAttribArray(a);
 gl.enableVertexAttribArray(sPos);gl.vertexAttribPointer(sPos,3,gl.FLOAT,false,32,0);gl.enableVertexAttribArray(sNor);gl.vertexAttribPointer(sNor,3,gl.FLOAT,false,32,12);
 gl.enableVertexAttribArray(sUv);gl.vertexAttribPointer(sUv,2,gl.FLOAT,false,32,24);
 gl.uniformMatrix4fv(sVP,false,vp);gl.uniform4f(sTint,...(tint||[0,0,0,0]));gl.activeTexture(gl.TEXTURE0);gl.uniform1i(sTex,0);
 for(const b of batches){const t=M.tex[b.tex];gl.bindTexture(gl.TEXTURE_2D,M.gltex[b.tex]);gl.uniform1f(sCut,t&&t.masked?1:0);gl.drawArrays(gl.TRIANGLES,b.start,b.n)}
 gl.disableVertexAttribArray(sPos);gl.disableVertexAttribArray(sNor);gl.disableVertexAttribArray(sUv)}
// ---- CS player: gait (legs) + aim (upper body) like the game's renderer ----
const CT_MODELS=['urban','gsg9','sas','gign','vip'],T_MODELS=['terror','leet','arctic','guerilla'];
function otherTeam(m){const i=CT_MODELS.indexOf(m);if(i>=0)return T_MODELS[Math.min(i,3)];const j=T_MODELS.indexOf(m);return j>=0?CT_MODELS[j]:'terror'}
function weaponExt(p){const w=(p||'').replace(/^p_/,'');
 if(w==='knife')return'knife';if(/^(usp|glock18|deagle|p228|fiveseven)$/.test(w))return'onehanded';if(w==='elite')return'dualpistols';
 if(/^(scout|awp|g3sg1|sg550)$/.test(w))return'rifle';if(w==='ak47')return'ak47';if(/^(m4a1|famas|galil)$/.test(w))return'rifle';if(w==='m249')return'm249';
 if(/^(m3|xm1014)$/.test(w))return'shotgun';if(/^(aug|sg552)$/.test(w))return'carbine';if(/grenade|flashbang/.test(w))return'grenade';if(w==='c4')return'c4';return w?'mp5':'onehanded'}
// st: {org:[x,y,z], yaw, pitch, ducked, onGround, speed, air (s since take-off), t (s)}
function drawPlayer(model,weapon,st,vp,tint){const M=mdlGet('player/'+model+'/'+model+'.mdl');if(!M)return false;
 const ext=weaponExt(weapon),pre=st.ducked?'crouch_aim_':'ref_aim_';let up=mdlSeq(M,pre+ext);if(up<0)up=mdlSeq(M,pre+'onehanded');if(up<0)up=0;
 let gait,gf;if(!st.onGround){gait=mdlSeq(M,'jump');gf=st.air*M.seqs[gait].fps}
 else if(st.ducked){gait=mdlSeq(M,st.speed>5?'crouchrun':'crouch_idle');const s=M.seqs[gait];gf=s.lin[0]>0?(st.dist/s.lin[0]*(s.frames-1)):st.t*s.fps}
 else{gait=mdlSeq(M,st.speed>135?'run':st.speed>5?'walk':'idle1');const s=M.seqs[gait];gf=s.lin[0]>0?(st.dist/s.lin[0]*(s.frames-1)):st.t*s.fps}
 if(gait<0)gait=0;const gs=M.seqs[gait],loop=st.onGround;gf=loop?gf%(gs.frames-1||1):Math.min(gf,gs.frames-1);
 const blend=Math.max(0,Math.min(1,(st.pitch+90)/180));
 const P=mdlPose(M,up,0,blend),G=mdlPose(M,gait,gf,0);
 // legs and pelvis from the gait sequence (bones before "Bip01 Spine")
 for(let b=0;b<M.bones.length;b++){if(M.bones[b].name==='Bip01 Spine')break;P.P[b]=G.P[b];P.Q[b]=G.Q[b]}
 const bones=mdlBones(M,P),y=st.yaw*Math.PI/180,X=[st.org[0],st.org[1],st.org[2],Math.cos(y),Math.sin(y),0,-Math.sin(y),Math.cos(y),0,0,0,1];
 mdlDraw(M,bones,X,vp,tint);
 if(weapon){const W=mdlGet(weapon+'.mdl');if(W){const map=new Map();M.bones.forEach((b,i)=>map.set(b.name,bones[i]));
   const wb=mdlBones(W,mdlPose(W,0,0,0),map);mdlDraw(W,wb,X,vp,tint)}}
 return true}
function drawViewModel(name,eye,pitch,yaw,t,vp){const M=mdlGet(name+'.mdl');if(!M)return;let s=mdlSeq(M,'idle');if(s<0)s=mdlSeq(M,'idle1');if(s<0)s=0;
 const q=M.seqs[s],f=(t*q.fps)%Math.max(1,q.frames-1),bones=mdlBones(M,mdlPose(M,s,f,0)),d=dirs(pitch,yaw,0);
 mdlDraw(M,bones,[eye[0],eye[1],eye[2],d.f[0],d.f[1],d.f[2],-d.r[0],-d.r[1],-d.r[2],d.u[0],d.u[1],d.u[2]],vp)}
