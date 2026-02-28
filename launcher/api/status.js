const https = require('https');

// Try to reach the hub and report its state
function probe(hostname) {
  return new Promise((resolve) => {
    const req = https.request({
      hostname,
      path: '/',
      method: 'HEAD',
      timeout: 8000,
      rejectUnauthorized: true,
    }, (res) => {
      resolve({ state: 'ready', code: res.statusCode });
    });

    req.on('timeout', () => { req.destroy(); resolve({ state: 'waiting' }); });
    req.on('error', (err) => {
      const msg = err.message || '';
      if (msg.includes('ECONNREFUSED') || msg.includes('ECONNRESET') || msg.includes('ETIMEDOUT') || msg.includes('ENOTFOUND')) {
        resolve({ state: 'waiting' });
      } else if (msg.includes('certificate') || msg.includes('cert') || msg.includes('SSL') || msg.includes('ERR_TLS')) {
        resolve({ state: 'tls_pending' });
      } else {
        resolve({ state: 'waiting' });
      }
    });

    req.end();
  });
}

module.exports = async function handler(req, res) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const hostname = req.query.hostname;
  if (!hostname || !/^[a-zA-Z0-9.-]+$/.test(hostname)) {
    return res.status(400).json({ error: 'Invalid hostname' });
  }

  const result = await probe(hostname);
  return res.status(200).json(result);
};
