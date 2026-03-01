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

  // 3. Once install reports done (step >= 4), check HTTPS for TLS status
  if (step >= 4) {
    const strict = await probeStrict(hostname);
    if (strict) {
      return res.status(200).json({ step: 6, logLine });
    }
    const relaxed = await probeRelaxed(hostname);
    if (relaxed) {
      return res.status(200).json({ step: 5, logLine });
    }
  }

  return res.status(200).json({ step, logLine });
};
