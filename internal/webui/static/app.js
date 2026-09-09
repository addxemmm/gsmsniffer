'use strict';
(() => {
  const $ = (s) => document.querySelector(s);
  const all = (s) => Array.from(document.querySelectorAll(s));
  const messages = {
    en: {
      workspace:'WORKSPACE',overview:'Overview',scan:'Frequency scan',capture:'Test capture',data:'Observations',api:'API reference',localLab:'Isolated laboratory',privacy:'Credentials remain only in this page’s memory.',theme:'Theme',shieldTitle:'SHIELDED ROOM / BOX ONLY',shieldBody:'Only test SIMs and devices you own or are authorized to use. Confirm effective physical shielding. Never operate in an open environment.',access:'Console access',accessNote:'Enter your server Bearer token. Reconnect after reloading this page.',connect:'Connect',disconnect:'Disconnect',systemOverview:'Laboratory overview',overviewHint:'Updates after connecting. Scans and captures never start automatically.',refresh:'Refresh',runtimeMode:'Runtime mode',activeTask:'Active task',singleTask:'See task status below',uptime:'Uptime',awaitConnection:'Awaiting connection',demoWarning:'DEMO · New tasks in the current mode generate synthetic data. Historical observations retain their individual source labels.',taskHistory:'Task history',autoRefresh:'Updates every 5 seconds · Pauses in background',connectFirst:'Connect to the server to view data.',capabilities:'Backend capabilities',scanTitle:'Create frequency observations',scanHint:'Choose a band and duration. Results appear in Observations.',band:'Band',duration:'Duration (seconds)',ack:'I confirm that I am using only owned or authorized SIMs / devices inside an effectively shielded room or shielded box.',startScan:'Start scan',captureTitle:'Configure shielded testing',captureHint:'Collect laboratory data from test SIMs / devices only. Confirm shielding before every start.',frequency:'Frequency (MHz)',observationType:'Observation type',startCapture:'Start test',observations:'Observations',dataHint:'Data may contain test identities or messages. Restrict access and clear it when no longer needed.',clearData:'Clear all observations',frequencies:'Frequencies',previous:'Previous',next:'Next',apiTitle:'Standard HTTP interface',apiHint:'Same-origin API v1 · JSON responses · Bearer authentication',authentication:'Authentication',endpoints:'Endpoints',apiStatus:'Runtime mode, version, active task and uptime.',apiCapabilities:'Query supported backend capabilities.',apiJobs:'List tasks.',apiStart:'Create a task; shielded_ack: true is required.',apiJob:'Get a single task.',apiStop:'Stop a task.',apiData:'Paginated frequencies, imsi or sms observations.',apiClear:'Explicitly delete all observations.',responseEnvelope:'Response envelope',apiSecurity:'Access through HTTPS or an SSH tunnel. Never commit tokens, real identities, messages or captured logs to public repositories or images.',footer:'Idle by default · Explicit start · Minimal data',offline:'Disconnected',online:'Connected',loading:'Loading…',empty:'No records yet.',idle:'Idle',synthetic:'Synthetic data only',shielded:'Shielded laboratory',failed:'Request failed',authRequired:'Enter a Bearer token first.',started:'Task created.',stopped:'Stop requested.',confirmStop:'Stop this task?',confirmClear:'Permanently delete ALL observations? This action cannot be undone.',cleared:'Observations cleared.',stop:'Stop',id:'Task ID',kind:'Kind',state:'State',startedAt:'Started',endedAt:'Ended',error:'Error',actions:'Actions',timestamp:'Time',arfcn:'ARFCN',frequency_mhz:'MHz',cell_id:'Cell ID',lac:'LAC',mcc:'MCC',mnc:'MNC',power_dbm:'dBm',identity:'Identity',text:'Message',total:'Total',unsafeToken:'Do not send credentials through remote plain HTTP. Use HTTPS or a local SSH tunnel.',ackRequired:'Confirm the shielded test conditions first.',taskError:'Task error',demoLabel:'DEMO / SYNTHETIC',source:'Source',sourceDemo:'Demo / synthetic',sourceShielded:'Shielded lab',sourceUnknown:'Unknown source'
    },
    zh: {offline:'未连接',online:'已连接',loading:'加载中…',empty:'暂无记录。',idle:'空闲',synthetic:'仅合成演示数据',shielded:'屏蔽实验环境',failed:'请求失败',authRequired:'请先输入 Bearer token 并连接。',started:'任务已创建。',stopped:'已请求停止任务。',confirmStop:'确认停止此任务？',confirmClear:'永久删除全部观测数据？此操作不可撤销。',cleared:'观测数据已清空。',stop:'停止',id:'任务 ID',kind:'类型',state:'状态',startedAt:'开始时间',endedAt:'结束时间',error:'错误',actions:'操作',timestamp:'时间',arfcn:'ARFCN',frequency_mhz:'MHz',cell_id:'小区 ID',lac:'LAC',mcc:'MCC',mnc:'MNC',power_dbm:'dBm',identity:'身份',text:'消息',total:'总计',unsafeToken:'请使用 HTTPS 或本地 SSH 隧道，避免通过远程明文 HTTP 发送凭据。',ackRequired:'请先确认屏蔽实验条件。',taskError:'任务错误',demoLabel:'演示 / 合成数据',source:'数据来源',sourceDemo:'演示 / 合成',sourceShielded:'屏蔽实验',sourceUnknown:'来源未知'}
  };
  all('[data-i18n]').forEach(el => { messages.zh[el.dataset.i18n] = el.textContent; });
  let lang = 'zh', token = '', panel = 'overview', connected = false, status = null, jobItems = [], observationData = null;
  let offset = 0, timer = null, refreshing = false, controller = null, generation = 0, mutating = false;
  const t = (key) => messages[lang][key] || key;
  const set = (selector, value) => { $(selector).textContent = value == null ? '—' : String(value); };
  const notify = (message, error = false) => { const el = $('#notice'); el.textContent = message; el.classList.remove('hidden'); el.classList.toggle('error', error); };
  const formatTime = (value) => { if (!value) return '—'; const d = new Date(value); return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-GB'); };
  function connection(value) { connected = value; set('#connection',t(value ? 'online' : 'offline')); $('#connection').classList.toggle('online',value); }
  async function api(path, options = {}) {
    if (!token) throw new Error(t('authRequired'));
    const requestController = new AbortController(), parentSignal = controller?.signal;
    const abort = () => requestController.abort();
    parentSignal?.addEventListener('abort',abort,{once:true});
    if(parentSignal?.aborted) abort();
    let timedOut = false;
    const timeout = setTimeout(() => {timedOut=true; abort();},15000);
    try {
      const response = await fetch('/api/v1' + path, { ...options, signal: requestController.signal, headers: { 'Authorization':'Bearer ' + token, ...(options.body ? {'Content-Type':'application/json'} : {}) }, cache:'no-store' });
      let envelope;
      try { envelope = await response.json(); } catch(error) { if(error.name==='AbortError') throw error; throw new Error(t('failed') + ' · HTTP ' + response.status); }
      if (!response.ok) { if (response.status === 401 || response.status === 403) connection(false); throw new Error((envelope.message || t('failed')) + (envelope.request_id ? ' [' + envelope.request_id + ']' : '')); }
      return envelope.data;
    } catch(error) { if(timedOut) throw new Error(t('failed')+' · 15s timeout'); throw error; }
    finally {clearTimeout(timeout); parentSignal?.removeEventListener('abort',abort);}
  }
  function table(container, columns, rows, action) {
    const target = $(container); target.replaceChildren(); target.classList.remove('empty');
    if (!rows.length) { target.classList.add('empty'); target.textContent = t('empty'); return; }
    const wrap = document.createElement('div'); wrap.className = 'table-wrap'; const tbl = document.createElement('table');
    const thead = document.createElement('thead'); const head = document.createElement('tr');
    columns.forEach(([key,label]) => { const th = document.createElement('th'); th.scope='col'; th.textContent=t(label || key); head.append(th); });
    if (action) {const th=document.createElement('th'); th.scope='col'; th.textContent=t('actions'); head.append(th);}
    thead.append(head); tbl.append(thead); const body=document.createElement('tbody');
    rows.forEach(row => { const tr=document.createElement('tr'); columns.forEach(([key]) => { const td=document.createElement('td'); const value=row[key]; td.textContent=key.endsWith('_at') || key === 'timestamp' ? formatTime(value) : value == null || value === '' ? '—' : String(value); tr.append(td); }); if(action) {const td=document.createElement('td'); action(td,row); tr.append(td);} body.append(tr); });
    tbl.append(body); wrap.append(tbl); target.append(wrap);
  }
  function renderStatus() {
    if (!status) return;
    const demo=status.mode==='demo'; set('#mode',demo ? 'DEMO' : String(status.mode || '—').toUpperCase()); set('#mode-note',t(demo ? 'synthetic':'shielded')); $('#demo-banner').classList.toggle('hidden',!demo);
    set('#active-job',status.active_job ? status.active_job.kind + ' / ' + status.active_job.state : t('idle'));
    const seconds=Math.max(0,Number(status.uptime_seconds)||0); set('#uptime',Math.floor(seconds/3600)+'h '+Math.floor(seconds%3600/60)+'m '+Math.floor(seconds%60)+'s'); set('#version','v'+(status.version || '—'));
  }
  function renderJobs() {
    table('#jobs',[['id'],['kind'],['state'],['started_at','startedAt'],['ended_at','endedAt'],['error']],jobItems,(td,job) => {
      if (!['pending','queued','running','starting','stopping'].includes(job.state)) return;
      const button=document.createElement('button'); button.className='secondary'; button.textContent=t('stop'); button.disabled=mutating || job.state==='stopping';
      button.addEventListener('click',() => mutate(async() => { if(!window.confirm(t('confirmStop'))) return; await api('/jobs/'+encodeURIComponent(job.id),{method:'DELETE'}); notify(t('stopped')); })); td.append(button);
    });
  }
  function renderData() {
    if (!observationData) return;
    const kind=$('#data-kind').value;
    const cols=kind==='frequencies' ? [['timestamp'],['arfcn'],['frequency_mhz'],['cell_id'],['lac'],['mcc'],['mnc'],['power_dbm']] : kind==='imsi' ? [['timestamp'],['identity'],['arfcn'],['frequency_mhz']] : [['timestamp'],['identity'],['text'],['arfcn'],['frequency_mhz']];
    cols.unshift(['source']);
    const rows=(observationData.items || []).map(row=>({...row,source:t(row.source==='demo'?'sourceDemo':row.source==='shielded'?'sourceShielded':'sourceUnknown')}));
    table('#observations',cols,rows); const total=Number(observationData.total)||0;
    set('#data-count',t('total')+' '+total); set('#page-index',Math.floor(offset/100)+1); $('#previous').disabled=offset===0; $('#next').disabled=offset+100>=total;
  }
  function schedule() { clearTimeout(timer); if(token && !document.hidden) timer=setTimeout(refresh,5000); }
  async function refresh() {
    if (!token || document.hidden || refreshing || mutating) return;
    refreshing=true; const epoch=generation; const currentOffset=offset, currentKind=$('#data-kind').value;
    $('#refresh').disabled=true;
    try {
      // Wait for every request to settle before scheduling another polling batch.
      const responses=await Promise.allSettled([api('/status'),api('/jobs'),api('/capabilities'),api('/observations?kind='+encodeURIComponent(currentKind)+'&limit=100&offset='+currentOffset)]);
      if(epoch!==generation) return;
      const failure=responses.find(result=>result.status==='rejected'); if(failure) throw failure.reason;
      const [newStatus,jobs,capabilities,observations]=responses.map(result=>result.value);
      status=newStatus; jobItems=jobs?.items || []; if(currentOffset===offset && currentKind===$('#data-kind').value) observationData=observations;
      connection(true); renderStatus(); renderJobs(); renderData(); set('#capabilities',JSON.stringify(capabilities,null,2));
      const maxDuration=Number(capabilities?.max_duration_seconds);
      if(Number.isInteger(maxDuration) && maxDuration>0) all('input[name=duration_seconds]').forEach(input=>{input.max=String(maxDuration);});
    } catch(error) { if(epoch===generation && error.name!=='AbortError') { connection(false); notify(error.message,true); } }
    finally { refreshing=false; $('#refresh').disabled=false; schedule(); }
  }
  async function mutate(work) {
    if(mutating) return;
    if(!token) {notify(t('authRequired'),true); return;}
    mutating=true; all('.job-form button[type=submit]').forEach(el=>el.disabled=true); $('#clear-data').disabled=true;
    try {await work();} catch(error) {if(error.name!=='AbortError') notify(error.message,true);} finally {mutating=false; all('.job-form button[type=submit]').forEach(el=>el.disabled=false); $('#clear-data').disabled=false; await refresh();}
  }
  all('[data-panel]').forEach(button => button.addEventListener('click',() => { panel=button.dataset.panel; all('.panel').forEach(el=>el.classList.toggle('hidden',el.id!=='panel-'+panel)); all('.nav').forEach(el=>el.classList.toggle('active',el===button)); set('#page-title',t(panel)); }));
  $('#language').addEventListener('click',() => {lang=lang==='zh'?'en':'zh'; document.documentElement.lang=lang==='zh'?'zh-CN':'en'; all('[data-i18n]').forEach(el=>el.textContent=t(el.dataset.i18n)); set('#language',lang==='zh'?'EN':'中文'); set('#page-title',t(panel)); connection(connected); renderStatus(); if(token) {renderJobs(); renderData();} });
  $('#theme').addEventListener('click',() => {document.documentElement.dataset.theme=document.documentElement.dataset.theme==='dark'?'light':'dark';});
  function disconnect() { generation++; clearTimeout(timer); controller?.abort(); controller=null; token=''; $('#token').value=''; status=null; jobItems=[]; observationData=null; connection(false); ['#mode','#active-job','#uptime','#version','#capabilities','#data-count'].forEach(s=>set(s,'—')); set('#mode-note',t('awaitConnection')); ['#jobs','#observations'].forEach(s=>{$(s).replaceChildren(); $(s).classList.add('empty'); set(s,t('connectFirst'));}); $('#demo-banner').classList.add('hidden'); offset=0; set('#page-index','1'); $('#previous').disabled=true; $('#next').disabled=true; all('.ack input').forEach(el=>el.checked=false); }
  $('#disconnect').addEventListener('click',() => {disconnect(); $('#notice').classList.add('hidden');});
  $('#auth-form').addEventListener('submit',async(event) => { event.preventDefault(); const value=$('#token').value.trim(); if(!value) return;
    if(location.protocol!=='https:' && !['localhost','127.0.0.1','[::1]'].includes(location.hostname)) {notify(t('unsafeToken'),true); return;}
    disconnect(); token=value; controller=new AbortController(); $('#notice').classList.add('hidden'); set('#jobs',t('loading')); set('#observations',t('loading')); await refresh();
  });
  all('.job-form').forEach(form => form.addEventListener('submit',event => {event.preventDefault(); if(!form.reportValidity()) return; const values=new FormData(form); if(!form.elements.shielded_ack.checked) {notify(t('ackRequired'),true); return;}
    const body={kind:form.dataset.kind,band:values.get('band'),duration_seconds:Number(values.get('duration_seconds')),shielded_ack:true};
    if(body.kind==='capture') {body.frequency_mhz=Number(values.get('frequency_mhz')); body.mode=values.get('mode');}
    mutate(async() => {try {await api('/jobs',{method:'POST',body:JSON.stringify(body)}); notify(t('started'));} finally {form.elements.shielded_ack.checked=false;}});
  }));
  function frequencyBand() {const dcs=$('#capture-form select[name=band]').value==='DCS1800', input=$('#capture-form input[name=frequency_mhz]'); input.min=dcs?'1805.2':'925.2'; input.max=dcs?'1879.8':'959.8'; input.value=dcs?'1845':'945';}
  $('#capture-form select[name=band]').addEventListener('change',frequencyBand); frequencyBand();
  $('#clear-data').addEventListener('click',()=>mutate(async()=>{if(!window.confirm(t('confirmClear'))) return; await api('/observations',{method:'DELETE'}); offset=0; notify(t('cleared'));}));
  $('#refresh').addEventListener('click',()=>{if(!token) notify(t('authRequired'),true); else refresh();});
  $('#data-kind').addEventListener('change',()=>{offset=0; observationData=null; set('#observations',token?t('loading'):t('connectFirst')); refresh();});
  $('#previous').addEventListener('click',()=>{offset=Math.max(0,offset-100); refresh();}); $('#next').addEventListener('click',()=>{offset+=100; refresh();});
  document.addEventListener('visibilitychange',()=>{if(document.hidden) clearTimeout(timer); else refresh();});
  window.addEventListener('pagehide',()=>{clearTimeout(timer); controller?.abort(); token='';});
  $('#previous').disabled=true; $('#next').disabled=true;
})();
