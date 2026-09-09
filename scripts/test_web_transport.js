'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync('internal/webui/static/app.js', 'utf8').split('(() => {')[0];
const context = {};
vm.runInNewContext(source, context);
const ip = (...octets) => octets.join('.');
const check = (hostname, expected, protocol = 'http:') =>
  assert.equal(context.allowTokenTransport({hostname, protocol}), expected, `${protocol}//${hostname}`);
for (const host of ['localhost', '127.0.0.1', '[::1]', ip(10,0,0,1), ip(10,255,255,255), ip(172,16,0,1), ip(172,31,255,255), ip(192,168,100,199)]) check(host, true);
for (const host of ['example.com', 'localhost.example.com', ip(8,8,8,8), ip(172,15,255,255), ip(172,32,0,1), ip(192,169,0,1), ip(10,0,0,256), '10.0.0', ip('010',0,0,1), ip(10,0,0,1) + '.example.com', '[fe80::1]', '']) check(host, false);
check('example.com', true, 'https:');
check(ip(10,0,0,1), false, 'file:');
console.log('Transport gate: private LAN/loopback HTTP accepted, public HTTP rejected, HTTPS accepted');
