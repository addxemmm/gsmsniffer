'use strict';
// Plain HTTP is an explicit trusted-LAN deployment choice, not encryption.
function allowTokenTransport(location) {
  if (location.protocol === 'https:') return true;
  if (location.protocol !== 'http:') return false;
  if (['localhost', '127.0.0.1', '[::1]'].includes(location.hostname)) return true;
  const parts = location.hostname.split('.');
  if (parts.length !== 4 || !parts.every(part => /^(0|[1-9][0-9]{0,2})$/.test(part) && Number(part) <= 255)) return false;
  const [a, b] = parts.map(Number);
  return a === 10 || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168);
}
(() => {
  const $ = (s) => document.querySelector(s);
  const all = (s) => Array.from(document.querySelectorAll(s));
  const messages = {
    en: {
      workspace:'WORKSPACE',overview:'Overview',scan:'Frequency scan',capture:'Test capture',data:'Observations',api:'API reference',localLab:'Isolated laboratory',privacy:'Credentials remain only in this page’s memory.',theme:'Theme',shieldTitle:'SHIELDED ROOM / BOX ONLY',shieldBody:'Only test SIMs and devices you own or are authorized to use. Confirm effective physical shielding. Never operate in an open environment.',access:'Console access',accessNote:'Enter your server Bearer token. HTTP sends it unencrypted: use only a trusted LAN. Reconnect after reloading.',connect:'Connect',disconnect:'Disconnect',systemOverview:'Laboratory overview',overviewHint:'Updates after connecting. Scans and captures never start automatically.',refresh:'Refresh',runtimeMode:'Runtime mode',activeTask:'Active task',singleTask:'See task status below',uptime:'Uptime',awaitConnection:'Awaiting connection',demoWarning:'DEMO · New tasks in the current mode generate synthetic data. Historical observations retain their individual source labels.',taskHistory:'Task history',autoRefresh:'Updates every 5 seconds · Pauses in background',connectFirst:'Connect to the server to view data.',capabilities:'Backend capabilities',scanTitle:'Create frequency observations',scanHint:'Choose a band and duration. Results appear in Observations.',band:'Band',duration:'Duration (seconds)',ack:'I confirm that I am using only owned or authorized SIMs / devices inside an effectively shielded room or shielded box.',startScan:'Start scan',captureTitle:'Configure shielded testing',captureHint:'Collect laboratory data from test SIMs / devices only. Confirm shielding before every start.',frequency:'Frequency (MHz)',observationType:'Observation type',startCapture:'Start test',observations:'Observations',dataHint:'Data may contain test identities or messages. Restrict access and clear it when no longer needed.',clearData:'Clear all observations',frequencies:'Frequencies',previous:'Previous',next:'Next',apiTitle:'Standard HTTP interface',apiHint:'Same-origin API v1 · JSON responses · Optional Bearer authentication',authentication:'Authentication',endpoints:'Endpoints',apiStatus:'Runtime mode, version, active task and uptime.',apiCapabilities:'Query supported backend capabilities.',apiJobs:'List tasks.',apiStart:'Create a task; shielded_ack: true is required.',apiJob:'Get a single task.',apiStop:'Stop a task.',apiData:'Paginated frequencies, imsi or sms observations.',apiClear:'Explicitly delete all observations.',responseEnvelope:'Response envelope',apiSecurity:'Use HTTPS, an SSH tunnel, or private-IP HTTP on a trusted LAN only. HTTP sends tokens unencrypted. Never commit tokens, real identities, messages or captured logs to public repositories or images.',footer:'Idle by default · Explicit start · Minimal data',offline:'Disconnected',online:'Connected',loading:'Loading…',empty:'No records yet.',idle:'Idle',synthetic:'Synthetic data only',shielded:'Shielded laboratory',failed:'Request failed',authRequired:'Enter a Bearer token first.',started:'Task created.',stopped:'Stop requested.',confirmStop:'Stop this task?',confirmClear:'Permanently delete ALL observations? This action cannot be undone.',cleared:'Observations cleared.',stop:'Stop',id:'Task ID',kind:'Kind',state:'State',startedAt:'Started',endedAt:'Ended',error:'Error',actions:'Actions',timestamp:'Time',arfcn:'ARFCN',frequency_mhz:'MHz',cell_id:'Cell ID',lac:'LAC',mcc:'MCC',mnc:'MNC',power_dbm:'dBm',identity:'Identity',text:'Message',total:'Total',unsafeToken:'HTTP login requires a private LAN IP or loopback. For other addresses, use HTTPS.',ackRequired:'Confirm the shielded test conditions first.',taskError:'Task error',demoLabel:'DEMO / SYNTHETIC',source:'Source',sourceDemo:'Demo / synthetic',sourceShielded:'Shielded lab',sourceUnknown:'Unknown source'
    },
    zh: {offline:'未连接',online:'已连接',loading:'加载中…',empty:'暂无记录。',idle:'空闲',synthetic:'仅合成演示数据',shielded:'屏蔽实验环境',failed:'请求失败',authRequired:'请先输入 Bearer token 并连接。',started:'任务已创建。',stopped:'已请求停止任务。',confirmStop:'确认停止此任务？',confirmClear:'永久删除全部观测数据？此操作不可撤销。',cleared:'观测数据已清空。',stop:'停止',id:'任务 ID',kind:'类型',state:'状态',startedAt:'开始时间',endedAt:'结束时间',error:'错误',actions:'操作',timestamp:'时间',arfcn:'ARFCN',frequency_mhz:'MHz',cell_id:'小区 ID',lac:'LAC',mcc:'MCC',mnc:'MNC',power_dbm:'dBm',identity:'身份',text:'消息',total:'总计',unsafeToken:'HTTP 登录仅支持局域网私有 IP 或本机地址；其他地址请使用 HTTPS。',ackRequired:'请先确认屏蔽实验条件。',taskError:'任务错误',demoLabel:'演示 / 合成数据',source:'数据来源',sourceDemo:'演示 / 合成',sourceShielded:'屏蔽实验',sourceUnknown:'来源未知'}
  };
  Object.assign(messages.en, {scan:'Scan workspace',workflowTitle:'Scan frequencies, then IMSI / SMS',workflowHint:'Choose a detected frequency; only one task runs at a time.',stepScan:'1 · Scan frequencies',scanHint:'Choose a band and scan. Results appear below; finish or stop scanning before selection.',stepSelect:'2 · Choose a detected frequency',selectHint:'Choose IMSI or SMS on a result row. Frequency, band and source scan are filled automatically.',scanBatch:'Scan batch',allScans:'All scan results',noFrequencies:'No usable frequencies yet. Run step 1 first.',stepCapture:'3 · Scan selected frequency for IMSI / SMS',selectFirst:'Select a frequency in step 2 first.',startScan:'Start frequency scan',startCapture:'Start IMSI / SMS scan',capturePrivacy:'IMSI remains masked; SMS is event-only without message bodies. No task starts automatically.',captureResults:'Results for selected frequency',waitingScan:'Waiting for scan to finish / stop',selectionExpired:'Selected frequency is no longer available. Scan again or select another result.',busyHint:'A task is running. Finish or stop it before starting another.',resultsLimit:'Most recent matching results',scanFailed:'Scan failed; check the task history and retry.'});
  Object.assign(messages.zh, {waitingScan:'等待频点扫描完成或停止',selectionExpired:'所选频点已失效，请重新扫描或选择其他结果。',busyHint:'已有任务运行，请等待完成或先停止。',resultsLimit:'最近的匹配结果',scanFailed:'频点扫描失败，请查看任务记录后重试。'});
  all('[data-i18n]').forEach(el => { messages.zh[el.dataset.i18n] = el.textContent; });
  let lang = 'zh', token = '', panel = 'scan', connected = false, status = null, jobItems = [], observationData = null;
  let frequencyItems = [], selectedFrequency = null, captureRows = [], frequencyRenderKey = "";
  const frequencyKey = row => row ? row.scan_job_id + ':' + row.frequency_mhz : '';
  let authRequired = true, authKnown = false;
  const canRequest = () => controller !== null && (!authRequired || token !== '');
  let offset = 0, timer = null, refreshing = false, controller = null, generation = 0, mutating = false;
  const t = (key) => messages[lang][key] || key;
  const set = (selector, value) => { $(selector).textContent = value == null ? '—' : String(value); };
  const notify = (message, error = false) => { const el = $('#notice'); el.textContent = message; el.classList.remove('hidden'); el.classList.toggle('error', error); };
  const formatTime = (value) => { if (!value) return '—'; const d = new Date(value); return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-GB'); };
  function connection(value) { connected = value; set('#connection',t(value ? 'online' : 'offline')); $('#connection').classList.toggle('online',value); }
  async function api(path, options = {}) {
    if (!canRequest()) throw new Error(t('authRequired'));
    const requestController = new AbortController(), parentSignal = controller?.signal;
    const abort = () => requestController.abort();
    parentSignal?.addEventListener('abort',abort,{once:true});
    if(parentSignal?.aborted) abort();
    let timedOut = false;
    const timeout = setTimeout(() => {timedOut=true; abort();},15000);
    try {
      const response = await fetch('/api/v1' + path, { ...options, signal: requestController.signal, headers: { ...(authRequired && token ? {'Authorization':'Bearer ' + token} : {}), ...(options.body ? {'Content-Type':'application/json'} : {}) }, cache:'no-store' });
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
    const demo=status.mode==='demo'; set('#mode',demo ? 'DEMO' : String(status.mode || '—').toUpperCase()); set('#mode-note',t(demo ? 'synthetic':'shielded')); $('#demo-banner').classList.toggle('hidden',!demo); $('#workflow-demo').classList.toggle('hidden',!demo);
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
  function workflowControls() {
    const busy = !!status?.active_job;
    $('#scan-form button[type=submit]').disabled = !canRequest() || !connected || busy || mutating;
    $('#capture-form button[type=submit]').disabled = !canRequest() || !connected || busy || mutating || !selectedFrequency?.selectable;
    $('#workflow-stop').disabled = !canRequest() || !busy || mutating;
    const job = status?.active_job;
    set('#workflow-job',job ? (job.kind === 'scan' ? t('stepScan') : String(job.config?.mode || '').toUpperCase() + ' · ' + job.config?.frequency_mhz + ' MHz') + ' · ' + job.state : t('idle'));
  }
  function renderCaptureRows() {
    if(!selectedFrequency) {set('#capture-results',t('selectFirst')); set('#capture-result-count','0'); return;}
    const mode = $('#capture-form select[name=mode]').value;
    const jobIDs = new Set(jobItems.filter(job => job.kind === 'capture' && job.config?.scan_job_id === selectedFrequency.scan_job_id && Number(job.config.frequency_mhz) === Number(selectedFrequency.frequency_mhz) && job.config.mode === mode).map(job => job.id));
    const rows = captureRows.filter(row => row.kind === mode && jobIDs.has(row.job_id)).slice().reverse();
    table('#capture-results',mode === 'imsi' ? [['timestamp'],['frequency_mhz'],['identity']] : [['timestamp'],['frequency_mhz'],['text']],rows);
    set('#capture-result-count',mode.toUpperCase() + ' · ' + rows.length);
  }
  function renderSelection() {
    const row = selectedFrequency;
    $('#capture-form input[name=band]').value = row?.band || '';
    $('#capture-form input[name=frequency_mhz]').value = row ? String(row.frequency_mhz) : '';
    $('#capture-form input[name=scan_job_id]').value = row?.scan_job_id || '';
    set('#selected-frequency',row ? row.band + ' · ' + row.frequency_mhz + ' MHz · ARFCN ' + row.arfcn + ' · ' + row.scan_job_id.slice(0,8) : t('selectFirst'));
    renderCaptureRows(); workflowControls();
  }
  function selectFrequency(row, mode) {
    if(!row.selectable || status?.active_job || mutating) return;
    selectedFrequency = row; captureRows = [];
    $('#capture-form select[name=mode]').value = mode;
    $('#capture-form input[name=shielded_ack]').checked = false;
    renderSelection();
    $('#capture-form').scrollIntoView({behavior:'smooth',block:'center'});
    refresh();
  }
  function renderFrequencies() {
    // Preserve result buttons and the open scan selector across unchanged polls.
    const nextKey=JSON.stringify([frequencyItems,lang,status?.active_job?.id,mutating,connected,$('#scan-filter').value,jobItems[0]?.state]);
    if(nextKey===frequencyRenderKey) {renderSelection();return;}
    frequencyRenderKey=nextKey;
    const filter = $('#scan-filter'), previous = filter.value;
    filter.replaceChildren();
    const allOption = document.createElement('option'); allOption.value=''; allOption.textContent=t('allScans'); filter.append(allOption);
    const batches = new Map();
    for(const row of frequencyItems) if(!batches.has(row.scan_job_id)) batches.set(row.scan_job_id,row);
    for(const [id,row] of batches) {const option=document.createElement('option'); option.value=id; option.textContent=row.band + ' · ' + formatTime(row.timestamp) + ' · ' + id.slice(0,8); filter.append(option);}
    filter.value=batches.has(previous)?previous:'';
    const rows=frequencyItems.filter(row=>!filter.value || row.scan_job_id===filter.value);
    set('#frequency-count',rows.length + ' / ' + frequencyItems.length);
    if(!rows.length) {const lastScan=jobItems.find(job=>job.kind==='scan');set('#frequency-results',t(lastScan?.state==='failed'?'scanFailed':'noFrequencies')); $('#frequency-results').classList.add('empty');}
    else table('#frequency-results',[['band'],['frequency_mhz'],['arfcn'],['mcc'],['mnc'],['power_dbm'],['source'],['timestamp']],rows,(td,row)=>{
      for(const mode of ['imsi','sms']) {const button=document.createElement('button'); button.className='secondary'; button.textContent=mode.toUpperCase(); button.disabled=!row.selectable || !!status?.active_job || mutating || !connected; button.title=row.selectable?t('selectHint'):t('waitingScan'); button.addEventListener('click',()=>selectFrequency(row,mode));td.append(button);}
    });
    renderSelection();
  }
  function schedule() { clearTimeout(timer); if(canRequest() && !document.hidden) timer=setTimeout(refresh,5000); }
  async function refresh() {
    if (!canRequest() || document.hidden || refreshing || mutating) return;
    refreshing=true; const epoch=generation; const currentOffset=offset, currentKind=$('#data-kind').value, currentSelection=frequencyKey(selectedFrequency), captureKind=$('#capture-form select[name=mode]').value;
    $('#refresh').disabled=true;
    try {
      // Wait for every request to settle before scheduling another polling batch.
      const responses=await Promise.allSettled([api('/status'),api('/jobs'),api('/capabilities'),api('/observations?kind='+encodeURIComponent(currentKind)+'&limit=100&offset='+currentOffset),api('/frequencies'),...(currentSelection ? [api('/observations?kind='+captureKind+'&limit=500&offset=0'),api('/observations?kind='+captureKind+'&limit=500&offset=500')] : [])]);
      if(epoch!==generation) return;
      const failure=responses.find(result=>result.status==='rejected'); if(failure) throw failure.reason;
      const [newStatus,jobs,capabilities,observations,frequencies,captureFirst,captureSecond]=responses.map(result=>result.value);
      status=newStatus; jobItems=jobs?.items || []; if(currentOffset===offset && currentKind===$('#data-kind').value) observationData=observations;
      frequencyItems=frequencies?.items || [];
      if(selectedFrequency) {
        const found=frequencyItems.find(row=>frequencyKey(row)===frequencyKey(selectedFrequency) && row.selectable);
        if(!found) {selectedFrequency=null; captureRows=[]; $('#capture-form input[name=shielded_ack]').checked=false; notify(t('selectionExpired'),true);} else selectedFrequency=found;
      }
      if(currentSelection===frequencyKey(selectedFrequency) && captureKind===$('#capture-form select[name=mode]').value) captureRows=[...(captureFirst?.items || []),...(captureSecond?.items || [])];
      connection(true); renderFrequencies(); renderStatus(); renderJobs(); renderData(); set('#capabilities',JSON.stringify(capabilities,null,2));
      const maxDuration=Number(capabilities?.max_duration_seconds);
      if(Number.isInteger(maxDuration) && maxDuration>0) all('input[name=duration_seconds]').forEach(input=>{input.max=String(maxDuration);});
    } catch(error) { if(epoch===generation && error.name!=='AbortError') { connection(false); notify(error.message,true); } }
    finally { refreshing=false; $('#refresh').disabled=false; workflowControls(); schedule(); }
  }
  async function mutate(work) {
    if(mutating) return;
    if(!canRequest()) {notify(t('authRequired'),true); return;}
    mutating=true; all('.job-form button[type=submit]').forEach(el=>el.disabled=true); $('#clear-data').disabled=true;
    try {await work();} catch(error) {if(error.name!=='AbortError') notify(error.message,true);} finally {mutating=false; $('#clear-data').disabled=false; workflowControls(); await refresh();}
  }
  all('[data-panel]').forEach(button => button.addEventListener('click',() => { panel=button.dataset.panel; all('.panel').forEach(el=>el.classList.toggle('hidden',el.id!=='panel-'+panel)); all('.nav').forEach(el=>el.classList.toggle('active',el===button)); set('#page-title',t(panel)); }));
  $('#language').addEventListener('click',() => {lang=lang==='zh'?'en':'zh'; document.documentElement.lang=lang==='zh'?'zh-CN':'en'; all('[data-i18n]').forEach(el=>el.textContent=t(el.dataset.i18n)); set('#language',lang==='zh'?'EN':'中文'); set('#page-title',t(panel)); connection(connected); renderStatus(); renderAuth(); if(canRequest()) {renderJobs(); renderData(); renderFrequencies();} });
  $('#theme').addEventListener('click',() => {document.documentElement.dataset.theme=document.documentElement.dataset.theme==='dark'?'light':'dark';});
  function disconnect() { generation++; clearTimeout(timer); controller?.abort(); controller=null; token=''; $('#token').value=''; status=null; jobItems=[]; observationData=null; frequencyItems=[]; selectedFrequency=null; captureRows=[]; connection(false); ['#mode','#active-job','#uptime','#version','#capabilities','#data-count'].forEach(s=>set(s,'—')); set('#mode-note',t('awaitConnection')); ['#jobs','#observations'].forEach(s=>{$(s).replaceChildren(); $(s).classList.add('empty'); set(s,t('connectFirst'));}); $('#demo-banner').classList.add('hidden'); $('#workflow-demo').classList.add('hidden'); offset=0; set('#page-index','1'); $('#previous').disabled=true; $('#next').disabled=true; all('.ack input').forEach(el=>el.checked=false); renderFrequencies(); }
  $('#disconnect').addEventListener('click',() => {disconnect(); $('#notice').classList.add('hidden');});
  $('#auth-form').addEventListener('submit',async(event) => { event.preventDefault(); if(!authKnown) {await bootstrapAuth(); return;} const value=$('#token').value.trim(); if(authRequired && !value) return;
    if(authRequired && !allowTokenTransport(location)) {notify(t('unsafeToken'),true); return;}
    disconnect(); token=value; controller=new AbortController(); $('#notice').classList.add('hidden'); set('#jobs',t('loading')); set('#observations',t('loading')); await refresh();
  });
  all('.job-form').forEach(form => form.addEventListener('submit',event => {event.preventDefault(); if(!form.reportValidity()) return; const values=new FormData(form); if(!form.elements.shielded_ack.checked) {notify(t('ackRequired'),true); return;}
    if(status?.active_job) {notify(t('busyHint'),true); return;}
    const body={kind:form.dataset.kind,band:values.get('band'),duration_seconds:Number(values.get('duration_seconds')),shielded_ack:true};
    if(body.kind==='capture') {
      if(!selectedFrequency?.selectable) {notify(t('selectFirst'),true); return;}
      body.scan_job_id=selectedFrequency.scan_job_id; body.band=selectedFrequency.band; body.frequency_mhz=selectedFrequency.frequency_mhz; body.mode=values.get('mode');
    }
    mutate(async() => {try {
      const job=await api('/jobs',{method:'POST',body:JSON.stringify(body)}); status={...status,active_job:job};
      if(body.kind==='scan') {selectedFrequency=null; captureRows=[]; $('#scan-filter').value='';}
      workflowControls(); notify(t('started'));
    } finally {form.elements.shielded_ack.checked=false;}});
  }));
  $('#capture-form select[name=mode]').addEventListener('change',()=>{captureRows=[];renderCaptureRows();refresh();});
  $('#scan-filter').addEventListener('change',renderFrequencies);
  $('#frequency-refresh').addEventListener('click',refresh);
  $('#workflow-stop').addEventListener('click',()=>mutate(async()=>{const id=status?.active_job?.id;if(!id || !window.confirm(t('confirmStop'))) return;await api('/jobs/'+encodeURIComponent(id),{method:'DELETE'});status.active_job=null;workflowControls();notify(t('stopped'));}));
  $('#clear-data').addEventListener('click',()=>mutate(async()=>{if(!window.confirm(t('confirmClear'))) return; await api('/observations',{method:'DELETE'}); offset=0; notify(t('cleared'));}));
  $('#refresh').addEventListener('click',()=>{if(!canRequest()) notify(t('authRequired'),true); else refresh();});
  $('#data-kind').addEventListener('change',()=>{offset=0; observationData=null; set('#observations',canRequest()?t('loading'):t('connectFirst')); refresh();});
  $('#previous').addEventListener('click',()=>{offset=Math.max(0,offset-100); refresh();}); $('#next').addEventListener('click',()=>{offset+=100; refresh();});
  document.addEventListener('visibilitychange',()=>{if(document.hidden) clearTimeout(timer); else refresh();});
  window.addEventListener('pagehide',()=>{clearTimeout(timer); controller?.abort(); token='';});
  function renderAuth() {
    $('#auth-form').classList.toggle('hidden', authKnown && !authRequired);
    $('#token').required = authKnown && authRequired;
    $('#token').disabled = !authKnown || !authRequired;
    if(authKnown && !authRequired) {
      $('[data-i18n="access"]').textContent = lang === 'zh' ? '免登录模式' : 'Anonymous access';
      $('[data-i18n="accessNote"]').textContent = lang === 'zh' ? '服务器未配置 Token，自动连接模式。能访问服务的人均可操作；仅限可信局域网。' : 'No token is configured. Automatic connection mode. Anyone who can reach this service can manage it; trusted LAN only.';
    }
  }
  async function bootstrapAuth() {
    const timeoutController = new AbortController();
    const timeout = setTimeout(() => timeoutController.abort(), 15000);
    try {
      const response = await fetch('/api/v1/auth', {cache:'no-store', signal:timeoutController.signal});
      const envelope = await response.json();
      if(!response.ok || envelope.code !== 'ok' || typeof envelope.data?.required !== 'boolean') throw new Error('Authentication configuration unavailable');
      authRequired = envelope.data.required; authKnown = true;
      renderAuth();
      if(!authRequired) {controller = new AbortController(); await refresh();}
    } catch(error) {notify(error.message || t('failed'),true);}
    finally {clearTimeout(timeout);}
  }
  $('#previous').disabled=true; $('#next').disabled=true;
  renderAuth();
  workflowControls();
  bootstrapAuth();
})();
