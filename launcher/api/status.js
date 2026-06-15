const http = require('http');
const https = require('https');
const { lunaRequest } = require('./_luna');

// Read install log from the status server on port 8080
function readInstallLog(ip) {
  return new Promise((resolve) => {
    const req = http.request({
      hostname: ip,
      port: 8080,
      path: '/',
      method: 'GET',
      timeout: 5000,
    }, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => resolve(data || null));
    });
    req.on('timeout', () => { req.destroy(); resolve(null); });
    req.on('error', () => resolve(null));
    req.end();
  });
}

// Map last timestamped log line to install step number
// 0=Creating VPS, 1=Booting, 2=Installing Docker, 3=Pulling images, 4=Starting services, 5=TLS, 6=Ready
function parseLogStep(line) {
  if (!line) return null;

  if (/running|finished/i.test(line)) return 4;
  if (/Starting containers/i.test(line)) return 4;
  if (/Pulling images/i.test(line)) return 3;
  if (/Docker installed|Docker already|Downloading|Writing .env/i.test(line)) return 3;
  if (/Installing Docker|Waiting for apt|Removing snap/i.test(line)) return 2;
  // Cloud-init just started (resolving IP, setting HOST_DOMAIN, etc.)
  return 2;
}

// TLS handshake completed but the certificate isn't trusted yet — i.e. Caddy
// is up and serving on 443 but the Let's Encrypt cert hasn't been issued.
// (vs. ECONNREFUSED/ETIMEDOUT/etc. which mean services aren't up at all)
const TLS_CERT_ERRORS = new Set([
  'UNABLE_TO_VERIFY_LEAF_SIGNATURE',
  'SELF_SIGNED_CERT_IN_CHAIN',
  'DEPTH_ZERO_SELF_SIGNED_CERT',
  'CERT_HAS_EXPIRED',
  'CERT_NOT_YET_VALID',
  'ERR_TLS_CERT_ALTNAME_INVALID',
  'UNABLE_TO_GET_ISSUER_CERT_LOCALLY',
]);

// Probe HTTPS with cert validation left ON (the default). A trusted cert means
// the hub is fully ready; a cert-validation error means services are up but the
// cert is still pending; anything else means services aren't responding yet.
function probe(hostname) {
  return new Promise((resolve) => {
    const req = https.request({
      hostname,
      path: '/',
      method: 'HEAD',
      timeout: 5000,
    }, () => {
      resolve('ready');
    });
    req.on('timeout', () => { req.destroy(); resolve(null); });
    req.on('error', (err) => {
      resolve(TLS_CERT_ERRORS.has(err.code) ? 'tls_pending' : null);
    });
    req.end();
  });
}

module.exports = async function handler(req, res) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { api_id, api_key, vm_id, hostname } = req.body || {};

  if (!hostname || !/^[a-zA-Z0-9.-]+$/.test(hostname)) {
    return res.status(400).json({ error: 'Invalid hostname' });
  }

  let step = 1; // default: booting
  let logLine = null;

  // 1. Get VM info (status + IP)
  let ip = null;
  if (api_id && api_key && vm_id) {
    try {
      const vmInfo = await lunaRequest(api_id, api_key, 'vm/info', { vm_id });
      const statusRaw = vmInfo.info && vmInfo.info.status_raw;
      ip = vmInfo.info && vmInfo.info.ip;

      // If VM not active yet, still booting
      if (statusRaw && statusRaw !== 'active') {
        return res.status(200).json({ step: 1, logLine });
      }
    } catch {
      // API call failed
    }
  }

  // 2. Try to read last log line via HTTP on port 8080
  if (ip) {
    const line = await readInstallLog(ip);
    if (line) {
      step = parseLogStep(line);
      logLine = line;
    }
    // Port 8080 not responding = cloud-init hasn't started our script yet
  }

  // 3. Check HTTPS for TLS status (always try — port 8080 may be unreachable mid-install)
  const tls = await probe(hostname);
  if (tls === 'ready') {
    return res.status(200).json({ step: 6, logLine });
  }
  if (tls === 'tls_pending') {
    return res.status(200).json({ step: 5, logLine });
  }

  return res.status(200).json({ step, logLine });
};
