const https = require('https');

// Probe with strict TLS (valid cert = fully ready)
function probeStrict(hostname) {
  return new Promise((resolve) => {
    const req = https.request({
      hostname,
      path: '/',
      method: 'HEAD',
      timeout: 8000,
      rejectUnauthorized: true,
    }, () => {
      resolve('ready');
    });
    req.on('timeout', () => { req.destroy(); resolve(null); });
    req.on('error', () => resolve(null));
    req.end();
  });
}

// Probe with relaxed TLS (any response = services running, cert may be pending)
function probeRelaxed(hostname) {
  return new Promise((resolve) => {
    const req = https.request({
      hostname,
      path: '/',
      method: 'HEAD',
      timeout: 8000,
      rejectUnauthorized: false,
    }, () => {
      resolve('tls_pending');
    });
    req.on('timeout', () => { req.destroy(); resolve(null); });
    req.on('error', () => resolve(null));
    req.end();
  });
}

async function probe(hostname) {
  const strict = await probeStrict(hostname);
  if (strict) return { state: strict };

  const relaxed = await probeRelaxed(hostname);
  if (relaxed) return { state: relaxed };

  return { state: 'waiting' };
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
