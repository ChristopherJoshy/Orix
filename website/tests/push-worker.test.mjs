import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
const handlers={},shown=[];
const self={addEventListener:(name,fn)=>handlers[name]=fn,
 registration:{showNotification:async(title,options)=>shown.push({title,...options})}};
vm.runInNewContext(fs.readFileSync(new URL('../public/orix-push-worker.js',import.meta.url),'utf8'),{self});
async function send(data){let pending;handlers.push({data:{json:()=>({data})},waitUntil:value=>pending=value});await pending;}
const notification_id='a'.repeat(64);
await send({type:'engagement_reminder',title:'Chaya break kazhinjo?',body:'Plan your next journey.',notification_id});
assert.equal(shown[0].title,'Chaya break kazhinjo?');assert.equal(shown[0].data.path,'/app');assert.equal(shown[0].silent,true);
await send({type:'inbox_update',notification_id});assert.equal(shown[1].title,'ORIX');assert.equal(shown[1].data.path,'/app/notifications');
await send({type:'engagement_reminder',notification_id,title:'x'.repeat(161),body:'invalid'});
await send({type:'engagement_reminder',notification_id:'not-an-id',title:'invalid',body:'invalid'});
assert.equal(shown.length,2);
console.log('Public reminder copy, private update privacy and malformed-push checks passed.');
