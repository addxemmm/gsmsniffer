'use strict';
// Exercise the real UI handlers against synthetic scan/capture API state only.
const assert=require('node:assert/strict'), fs=require('node:fs'), vm=require('node:vm');
const nodes=new Map(), calls=[], state={active:null,jobs:[],frequencies:[],observations:[]};
function element(key='') {
  const classes=new Set();
  return {key,children:[],value:'',textContent:'',dataset:{},events:{},checked:false,disabled:false,
    classList:{add:c=>classes.add(c),remove:c=>classes.delete(c),toggle(c,v){if(v)classes.add(c);else classes.delete(c);},contains:c=>classes.has(c)},
    append(...children){this.children.push(...children);},replaceChildren(...children){this.children=[...children];},
    addEventListener(event,fn){this.events[event]=fn;},scrollIntoView(){},reportValidity(){return true;}};
}
function node(key){if(!nodes.has(key))nodes.set(key,element(key));return nodes.get(key);}
for(const kind of ['scan','capture']){
  const form=node('#'+kind+'-form');form.dataset.kind=kind;
  form.elements={shielded_ack:node('#'+kind+'-form input[name=shielded_ack]')};
  node('#'+kind+'-form input[name=duration_seconds]').value='1';
}
node('#scan-form select[name=band]').value='GSM900';
node('#capture-form select[name=mode]').value='imsi';node('#data-kind').value='frequencies';
const document={hidden:false,documentElement:{dataset:{}},querySelector:node,createElement:()=>element(),addEventListener(){},
  querySelectorAll(selector){
    if(selector==='.job-form')return [node('#scan-form'),node('#capture-form')];
    if(selector==='.job-form button[type=submit]')return [node('#scan-form button[type=submit]'),node('#capture-form button[type=submit]')];
    if(selector==='.ack input')return [node('#scan-form').elements.shielded_ack,node('#capture-form').elements.shielded_ack];
    return [];
  }};
class FormData {
  constructor(form){this.form=form;}
  get(key){const type=key==='mode'||(key==='band'&&this.form.dataset.kind==='scan')?'select':'input';return node(this.form.key+' '+type+'[name='+key+']').value;}
}
async function fetch(url,options={}){
  calls.push({url,options});let data={};
  if(url.endsWith('/auth'))data={required:false};
  else if(url.endsWith('/status'))data={mode:'demo',version:'2.1',active_job:state.active};
  else if(url.endsWith('/capabilities'))data={max_duration_seconds:300};
  else if(url.endsWith('/frequencies'))data={items:state.frequencies,total:state.frequencies.length};
  else if(url.endsWith('/jobs')&&options.method==='POST'){
    const config=JSON.parse(options.body);const job={id:String(state.jobs.length+1).padStart(32,'a'),kind:config.kind,state:'running',config};
    state.active=job;state.jobs.unshift(job);data=job;
  }else if(url.includes('/jobs/')&&options.method==='DELETE'){
    const job=state.jobs.find(j=>url.endsWith(j.id));job.state='cancelled';state.active=null;
    if(job.kind==='scan')state.frequencies.forEach(row=>{row.selectable=true;});data=job;
  }else if(url.endsWith('/jobs'))data={items:state.jobs};
  else if(options.method==='DELETE'&&url.endsWith('/observations')){state.frequencies=[];state.observations=[];}
  else if(url.includes('/observations'))data={items:state.observations,total:state.observations.length};
  return {ok:true,status:200,json:async()=>({code:'ok',data:structuredClone(data)})};
}
vm.runInNewContext(fs.readFileSync('internal/webui/static/app.js','utf8'),{
  document,window:{addEventListener(){},confirm:()=>true},location:{protocol:'http:',hostname:'localhost'},fetch,FormData,AbortController,setTimeout:()=>1,clearTimeout(){},console
});
const tick=()=>new Promise(resolve=>setImmediate(resolve));
const posts=()=>calls.filter(c=>c.options.method==='POST');
async function click(key,event='click'){await node(key).events[event]({preventDefault(){}});await tick();}
function buttons(root){return root.children.flatMap(child=>[child,...buttons(child)]).filter(child=>child.events.click);}
async function select(mode){const button=buttons(node('#frequency-results')).find(b=>b.textContent===mode);assert.ok(button);assert.equal(button.disabled,false);button.events.click();await tick();}
(async()=>{
  await tick();assert.equal(posts().length,0);assert.equal(node('#capture-form button[type=submit]').disabled,true);
  node('#scan-form').elements.shielded_ack.checked=true;await click('#scan-form','submit');
  assert.equal(posts().length,1);assert.equal(JSON.parse(posts()[0].options.body).kind,'scan');
  const scan=state.jobs[0];state.frequencies=[{scan_job_id:scan.id,band:'GSM900',frequency_mhz:935.2,arfcn:1,source:'demo',timestamp:new Date().toISOString(),selectable:false}];
  await click('#frequency-refresh');assert.ok(buttons(node('#frequency-results')).every(b=>b.disabled));
  await click('#workflow-stop');await select('IMSI');
  assert.equal(posts().length,1,'Selection must not start a task');
  assert.equal(node('#capture-form input[name=frequency_mhz]').value,'935.2');
  assert.equal(node('#capture-form input[name=scan_job_id]').value,scan.id);
  node('#capture-form').elements.shielded_ack.checked=true;await click('#capture-form','submit');
  const capture=JSON.parse(posts()[1].options.body);assert.equal(capture.scan_job_id,scan.id);assert.equal(capture.frequency_mhz,935.2);assert.equal(capture.mode,'imsi');
  assert.equal(node('#scan-form button[type=submit]').disabled,true);
  await click('#workflow-stop');await select('SMS');node('#capture-form').elements.shielded_ack.checked=true;await click('#capture-form','submit');
  assert.equal(JSON.parse(posts()[2].options.body).mode,'sms');assert.equal(JSON.parse(posts()[2].options.body).scan_job_id,scan.id);
  await click('#workflow-stop');await click('#clear-data');
  assert.equal(node('#capture-form input[name=frequency_mhz]').value,'');assert.equal(node('#capture-form button[type=submit]').disabled,true);
  assert.equal(state.active,null);
  console.log('Workflow passed: scan -> stop -> choose IMSI -> capture -> stop -> choose SMS -> capture -> clear invalidates selection; no automatic tasks');
})().catch(error=>{console.error(error);process.exitCode=1;});
