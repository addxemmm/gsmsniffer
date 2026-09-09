'use strict';
// Run the real console script with a minimal DOM and synthetic API responses.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
async function consoleFixture(required, failed = false) {
  const nodes = new Map(), calls = [];
  function node(key) {
    if (!nodes.has(key)) {
      const classes = new Set();
      nodes.set(key, {value: key === '#data-kind' ? 'frequencies' : '', textContent:'', dataset:{}, events:{},
        classList:{add:c=>classes.add(c), remove:c=>classes.delete(c), contains:c=>classes.has(c), toggle(c,on){if(on) classes.add(c); else classes.delete(c);}},
        addEventListener(name,handler){this.events[name]=handler;}, replaceChildren(){}, append(){}});
    }
    return nodes.get(key);
  }
  const document = {hidden:false, documentElement:{dataset:{}}, querySelector:node, querySelectorAll:()=>[], createElement:()=>node(Symbol()), addEventListener(){}};
  const fetch = async (url, options={}) => {
    calls.push({url, options});
    if(failed) throw new Error('metadata unavailable');
    const data = url.endsWith('/auth') ? {required} : url.endsWith('/status') ? {mode:'demo',version:'2.1',uptime_seconds:1} : url.endsWith('/capabilities') ? {max_duration_seconds:300} : {items:[],total:0};
    return {ok:true, status:200, json:async()=>({code:'ok',data})};
  };
  vm.runInNewContext(fs.readFileSync('internal/webui/static/app.js','utf8'), {
    document, window:{addEventListener(){}}, location:{protocol:'http:',hostname:[192,168,1,5].join('.')},
    fetch, AbortController, setTimeout:()=>1, clearTimeout(){}, console
  });
  await new Promise(resolve=>setImmediate(resolve));
  return {node,calls};
}
(async()=>{
  let f=await consoleFixture(false);
  assert.equal(f.node('#auth-form').classList.contains('hidden'),true);
  assert.equal(f.node('#token').required,false);
  assert.equal(f.node('#connection').textContent,'已连接');
  assert.equal(f.calls.length,5);
  assert.ok(f.calls.every(c=>!c.options.headers?.Authorization));
  await f.node('#language').events.click();
  assert.equal(f.node('[data-i18n="access"]').textContent,'Anonymous access');
  f=await consoleFixture(true);
  assert.equal(f.calls.length,1); // Do not call protected APIs before login.
  assert.equal(f.node('#token').required,true);
  assert.equal(f.node('#auth-form').classList.contains('hidden'),false);
  f.node('#token').value='synthetic-test-token-32-characters-only';
  await f.node('#auth-form').events.submit({preventDefault(){}});
  assert.equal(f.node('#connection').textContent,'已连接');
  assert.ok(f.calls.slice(1).every(c=>c.options.headers.Authorization.startsWith('Bearer ')));
  f=await consoleFixture(true,true);
  assert.equal(f.calls.length,1);
  assert.equal(f.node('#token').disabled,true);
  assert.equal(f.node('#auth-form').classList.contains('hidden'),false);
  console.log('Console optional auth: anonymous auto-connect, token-required login and metadata failure passed');
})().catch(error=>{console.error(error);process.exitCode=1;});
