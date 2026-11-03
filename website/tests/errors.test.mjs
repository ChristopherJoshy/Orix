function check(name,run){run();console.log(`PASS ${name}`);}
import assert from 'node:assert/strict';
import { PublicError, friendlyError, statusMessage } from '/tmp/orix-web-tests/errors.js';

check('technical and private error details never become interface copy',()=>{
  for(const message of ['MongoServerError: private-host','Bearer private-token','/home/private/path','TypeError: stack-trace']){
    const result=friendlyError(new Error(message));assert.ok(!result.includes(message));assert.ok(result.includes('try'));
  }
  assert.ok(friendlyError(new TypeError('Failed to fetch')).includes('chaya'));
});
check('only authored public validation messages are preserved',()=>{
  assert.equal(friendlyError(new PublicError('Choose a future departure time.')),'Choose a future departure time.');
  assert.equal(friendlyError({message:'database-password'}),'That didn’t go through. Let’s give it another try.');
});
check('authentication and safety errors remain clear, technical failures stay private',()=>{
  assert.ok(statusMessage(401).includes('sign in'));
  assert.ok(statusMessage(403).includes('account'));
  assert.ok(statusMessage(409).includes('Refresh'));
  assert.ok(statusMessage(429).includes('Wait'));
  assert.ok(statusMessage(503).includes('chaya'));
});
