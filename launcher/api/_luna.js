const crypto = require('crypto');
const https = require('https');

// LunaNode API: HMAC-SHA512 signed requests
function signRequest(apiId, apiKey, handler, params) {
  const partialKey = apiKey.substring(0, 64);
  const body = JSON.stringify({ api_id: apiId, api_partialkey: partialKey, ...params });
  const nonce = Math.floor(Date.now() / 1000).toString();
  const signature = crypto
    .createHmac('sha512', apiKey)
    .update(`${handler}/|${body}|${nonce}`)
    .digest('hex');
  return { body, signature, nonce };
}

function lunaRequest(apiId, apiKey, handler, params = {}) {
  return new Promise((resolve, reject) => {
    const { body, signature, nonce } = signRequest(apiId, apiKey, handler, params);
    const postData = `req=${encodeURIComponent(body)}&signature=${encodeURIComponent(signature)}&nonce=${encodeURIComponent(nonce)}`;

    const req = https.request({
      hostname: 'dynamic.lunanode.com',
      path: `/api/${handler}/`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'Content-Length': Buffer.byteLength(postData),
      },
    }, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          if (json.success !== 'yes') {
            reject(new Error(json.error || `API error on ${handler}`));
          } else {
            resolve(json);
          }
        } catch {
          reject(new Error(`Invalid response from ${handler}`));
        }
      });
    });

    req.on('error', reject);
    req.setTimeout(15000, () => { req.destroy(); reject(new Error(`Timeout on ${handler}`)); });
    req.write(postData);
    req.end();
  });
}

module.exports = { lunaRequest };
