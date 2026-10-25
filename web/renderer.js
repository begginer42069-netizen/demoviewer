function parseBSP(buf){
 const dv=new DataView(buf),u8=new Uint8Array(buf);
 if(dv.getInt32(0,true)!==30)throw new Error('not a GoldSrc BSP (version '+dv.getInt32(0,true)+')');
 const L=[];for(let i=0;i<15;i++)L.push({o:dv.getInt32(4+i*8,true),l:dv.getInt32(8+i*8,true)});
 const f32=(o)=>dv.getFloat32(o,true),i32=o=>dv.getInt32(o,true),u16=o=>dv.getUint16(o,true);
 // entities
 const entText=new TextDecoder('latin1').decode(u8.subarray(L[0].o,L[0].o+L[0].l)).replace(/\0.*$/,'');
 const ents=[];entText.replace(/\{([^}]*)\}/g,(m,b)=>{const e={};b.replace(/"([^"]*)"\s+"([^"]*)"/g,(mm,k,v)=>{e[k]=v});ents.push(e)});
 // textures
 const to=L[2].o,nt=i32(to),tex=[];
 for(let i=0;i<nt;i++){const off=i32(to+4+i*4);if(off<0){tex.push(null);continue}
  const b=to+off;let name='';for(let k=0;k<16;k++){const c=u8[b+k];if(!c)break;name+=String.fromCharCode(c)}
  const w=dv.getUint32(b+16,true),h=dv.getUint32(b+20,true),o0=dv.getUint32(b+24,true);
  if(!o0){tex.push({name,w,h,rgba:null});continue}
  const px=b+o0,pal=b+dv.getUint32(b+36,true)+((w>>3)*(h>>3))+2;
  const rgba=new Uint8Array(w*h*4);const trans=name[0]==='{';
  for(let k=0;k<w*h;k++){const ix=u8[px+k];rgba[k*4]=u8[pal+ix*3];rgba[k*4+1]=u8[pal+ix*3+1];rgba[k*4+2]=u8[pal+ix*3+2];rgba[k*4+3]=(trans&&ix===255)?0:255}
  tex.push({name,w,h,rgba,trans})}
 const vo=L[3].o,eo=L[12].o,seo=L[13].o,fo=L[7].o,tio=L[6].o,mo=L[14].o,lo=L[8].o,ll=L[8].l;
 const nfaces=L[7].l/20,nmodels=L[14].l/64;
 const vert=i=>[f32(vo+i*12),f32(vo+i*12+4),f32(vo+i*12+8)];
 const faces=[];
 for(let i=0;i<nfaces;i++){const b=fo+i*20;
  const first=i32(b+4),num=u16(b+8),ti=u16(b+10),lofs=i32(b+16);
  const tb=tio+ti*40;const S=[f32(tb),f32(tb+4),f32(tb+8)],oS=f32(tb+12),T=[f32(tb+16),f32(tb+20),f32(tb+24)],oT=f32(tb+28),mt=i32(tb+32),flags=i32(tb+36);
  const vs=[];for(let k=0;k<num;k++){const se=i32(seo+(first+k)*4);const e=Math.abs(se),vi=u16(eo+e*4+(se<0?2:0));vs.push(vert(vi))}
  faces.push({vs,S,oS,T,oT,mt,flags,lofs})}
 const models=[];for(let i=0;i<nmodels;i++){const b=mo+i*64;models.push({first:i32(b+56),num:i32(b+60),origin:[f32(b+24),f32(b+28),f32(b+32)]})}
 return {ents,tex,faces,models,lighting:u8.subarray(lo,lo+ll)};
}

const SKIP=/^(sky|clip|null|aaatrigger|origin|skip|hint|trigger|noclip|bevel)/i;
function rotB(v,a){const D=Math.PI/180;let[x,y,z]=v;if(a[2]){const c=Math.cos(a[2]*D),s=Math.sin(a[2]*D);[y,z]=[y*c-z*s,y*s+z*c]}
 if(a[0]){const c=Math.cos(-a[0]*D),s=Math.sin(-a[0]*D);[x,z]=[x*c+z*s,-x*s+z*c]}if(a[1]){const c=Math.cos(a[1]*D),s=Math.sin(a[1]*D);[x,y]=[x*c-y*s,x*s+y*c]}return[x,y,z]}
function buildScene(B){
 // decide which models are drawn and with which offset / alpha
 const draw=[{m:0,off:[0,0,0],alpha:1}];
 for(const e of B.ents){const m=/^\*(\d+)$/.exec(e.model||'');if(!m)continue;const c=e.classname||'';
  if(/^(trigger_|func_ladder|func_buyzone|func_bomb|func_hostage|func_vip|func_monster|func_clip|func_friction|func_water|func_conveyor)/.test(c))continue;
  const rm=+(e.rendermode||0),amt=+(e.renderamt||255);if(rm!==0&&amt<=0)continue;
  const ov=window.ENT_OVR&&window.ENT_OVR[m[1]];const o=ov?ov.slice(0,3):(e.origin||'0 0 0').split(' ').map(Number);
  draw.push({m:+m[1],off:o,ang:ov&&(ov[3]||ov[4]||ov[5])?ov.slice(3,6):null,alpha:rm===0||rm===4?1:Math.max(0.15,amt/255)})}
 // lightmap atlas
 const AW=2048,AH=2048,atlas=new Uint8Array(AW*AH*4);for(let i=0;i<AW*AH;i++){atlas[i*4]=atlas[i*4+1]=atlas[i*4+2]=255;atlas[i*4+3]=255}
 let px=0,py=0,rowh=0;const place=(w,h,data)=>{if(px+w+1>AW){px=0;py+=rowh+1;rowh=0}if(py+h>AH)return null;
  const x0=px,y0=py;for(let y=0;y<h;y++)for(let x=0;x<w;x++){const s=(y*w+x)*3,d=((y0+y)*AW+x0+x)*4;atlas[d]=data[s];atlas[d+1]=data[s+1];atlas[d+2]=data[s+2]}
  px+=w+1;rowh=Math.max(rowh,h);return [x0,y0]};
 const batches=new Map();// key texIndex|alpha -> arrays
 const faceLM=new Map();
 for(const d of draw){const M=B.models[d.m];if(!M)continue;
  for(let fi=M.first;fi<M.first+M.num;fi++){const F=B.faces[fi];const tx=B.tex[F.mt];if(!tx||!tx.rgba||SKIP.test(tx.name))continue;
   const us=[],vs=[];for(const v of F.vs){us.push(v[0]*F.S[0]+v[1]*F.S[1]+v[2]*F.S[2]+F.oS);vs.push(v[0]*F.T[0]+v[1]*F.T[1]+v[2]*F.T[2]+F.oT)}
   let lm=null;const special=(F.flags&1)||tx.name[0]==='!';
   if(!special&&F.lofs>=0){const smin=Math.floor(Math.min(...us)/16),smax=Math.ceil(Math.max(...us)/16),tmin=Math.floor(Math.min(...vs)/16),tmax=Math.ceil(Math.max(...vs)/16);
    const w=smax-smin+1,h=tmax-tmin+1;
    if(F.lofs+w*h*3<=B.lighting.length&&w<=128&&h<=128){const key=fi;let p=faceLM.get(key);if(!p){p=place(w,h,B.lighting.subarray(F.lofs,F.lofs+w*h*3));faceLM.set(key,p)}
     if(p)lm={x:p[0],y:p[1],w,h,smin,tmin}}}
   const mv=window.ENT_MOV&&window.ENT_MOV[d.m]?d.m:0;const key=B.tex.indexOf(tx)+'|'+d.alpha+'|'+mv;let bt=batches.get(key);if(!bt){bt={tex:tx,alpha:d.alpha,data:[],water:tx.name[0]==='!',mov:mv};batches.set(key,bt)}
   const vtx=F.vs.map((v,k)=>{let lu=-1,lv=-1;if(lm){lu=((us[k]/16-lm.smin)+0.5+lm.x)/AW;lv=((vs[k]/16-lm.tmin)+0.5+lm.y)/AH}
    const w=d.ang?rotB(v,d.ang):v;return [w[0]+d.off[0],w[1]+d.off[1],w[2]+d.off[2],us[k]/tx.w,vs[k]/tx.h,lu,lv]});
   for(let k=1;k<vtx.length-1;k++)for(const q of [vtx[0],vtx[k],vtx[k+1]])bt.data.push(...q)}}
 return {batches:[...batches.values()],atlas,AW,AH};
}

let prog,aPos,aUv,aLm,uMoff,uVP,uTex,uLm,uAlpha,uWater,uCut,lmTex,glb=[];
function initGL(){
 const vs=`#version 300 es
in vec3 p;in vec2 uv;in vec2 lm;uniform mat4 vp;uniform vec3 moff;out vec2 vuv;out vec2 vlm;
void main(){vuv=uv;vlm=lm;gl_Position=vp*vec4(p+moff,1.0);}`;
 const fs=`#version 300 es
precision highp float;in vec2 vuv;in vec2 vlm;uniform sampler2D tex;uniform sampler2D lmt;uniform float alpha;uniform float water;uniform float cut;out vec4 o;
void main(){vec4 c=texture(tex,vuv);if(c.a<0.5&&cut>0.5)discard;vec3 l=vlm.x<0.0?vec3(1.0):texture(lmt,vlm).rgb;
 vec3 col=c.rgb*l*2.0;if(water>0.5){col=c.rgb*0.8+vec3(0.0,0.05,0.15);}
 col=pow(clamp(col,0.0,1.0),vec3(0.85));o=vec4(col,water>0.5?0.55:alpha);}`;
 const sh=(t,s)=>{const x=gl.createShader(t);gl.shaderSource(x,s);gl.compileShader(x);if(!gl.getShaderParameter(x,gl.COMPILE_STATUS))throw new Error(gl.getShaderInfoLog(x));return x};
 prog=gl.createProgram();gl.attachShader(prog,sh(gl.VERTEX_SHADER,vs));gl.attachShader(prog,sh(gl.FRAGMENT_SHADER,fs));gl.linkProgram(prog);
 if(!gl.getProgramParameter(prog,gl.LINK_STATUS))throw new Error(gl.getProgramInfoLog(prog));
 aPos=gl.getAttribLocation(prog,'p');aUv=gl.getAttribLocation(prog,'uv');aLm=gl.getAttribLocation(prog,'lm');
 uVP=gl.getUniformLocation(prog,'vp');uMoff=gl.getUniformLocation(prog,'moff');uTex=gl.getUniformLocation(prog,'tex');uLm=gl.getUniformLocation(prog,'lmt');uAlpha=gl.getUniformLocation(prog,'alpha');uWater=gl.getUniformLocation(prog,'water');uCut=gl.getUniformLocation(prog,'cut');
}
function upload(S){
 glb.forEach(b=>{gl.deleteBuffer(b.buf);gl.deleteTexture(b.t)});glb=[];
 if(lmTex)gl.deleteTexture(lmTex);lmTex=gl.createTexture();gl.bindTexture(gl.TEXTURE_2D,lmTex);
 gl.texImage2D(gl.TEXTURE_2D,0,gl.RGBA,S.AW,S.AH,0,gl.RGBA,gl.UNSIGNED_BYTE,S.atlas);
 gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MIN_FILTER,gl.LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MAG_FILTER,gl.LINEAR);
 gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_S,gl.CLAMP_TO_EDGE);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_T,gl.CLAMP_TO_EDGE);
 for(const b of S.batches){const t=gl.createTexture();gl.bindTexture(gl.TEXTURE_2D,t);
  gl.texImage2D(gl.TEXTURE_2D,0,gl.RGBA,b.tex.w,b.tex.h,0,gl.RGBA,gl.UNSIGNED_BYTE,b.tex.rgba);gl.generateMipmap(gl.TEXTURE_2D);
  gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MIN_FILTER,gl.LINEAR_MIPMAP_LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MAG_FILTER,gl.LINEAR);
  gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_S,gl.REPEAT);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_T,gl.REPEAT);
  const buf=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,buf);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array(b.data),gl.STATIC_DRAW);
  glb.push({buf,t,n:b.data.length/7,alpha:b.alpha,water:b.water,cut:b.tex.trans?1:0,mov:b.mov})}
}
function dirs(p,y,r){const d=Math.PI/180,sp=Math.sin(p*d),cp=Math.cos(p*d),sy=Math.sin(y*d),cy=Math.cos(y*d),sr=Math.sin(r*d),cr=Math.cos(r*d);
 return {f:[cp*cy,cp*sy,-sp],r:[-sr*sp*cy+cr*sy,-sr*sp*sy-cr*cy,-sr*cp],u:[cr*sp*cy-sr*sy,cr*sp*sy+sr*cy,cr*cp]}}
function viewProj(eye,p,y,r,aspect){const D=dirs(p,y,r),dot=(a,b)=>a[0]*b[0]+a[1]*b[1]+a[2]*b[2];
 const R=D.r,U=D.u,F=D.f;
 const V=[R[0],U[0],-F[0],0, R[1],U[1],-F[1],0, R[2],U[2],-F[2],0, -dot(R,eye),-dot(U,eye),dot(F,eye),1];
 const vf=2*Math.atan(Math.tan(fov*Math.PI/360)*3/4),f=1/Math.tan(vf/2),n=1,fa=12000;
 const P=[f/aspect,0,0,0, 0,f,0,0, 0,0,(fa+n)/(n-fa),-1, 0,0,2*fa*n/(n-fa),0];
 const o=new Array(16).fill(0);for(let c=0;c<4;c++)for(let r2=0;r2<4;r2++){let s=0;for(let k=0;k<4;k++)s+=P[k*4+r2]*V[c*4+k];o[c*4+r2]=s}return o}

