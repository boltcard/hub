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

const STARTUP_SCRIPT = `#!/bin/bash
export HOME="\${HOME:-/root}"
export DEBIAN_FRONTEND=noninteractive

# Resolve rDNS hostname from public IP
IP=$(curl -4 -s --retry 5 --retry-delay 2 ifconfig.me)
HOST_DOMAIN=$(host "$IP" 2>/dev/null | awk '/domain name pointer/ {sub(/[.]$/, "", $NF); print $NF}') || true

if [ -z "$HOST_DOMAIN" ]; then
  IFS='.' read -r a b c d <<< "$IP"
  HOST_DOMAIN="$d.$c.$b.$a.lunanode-rdns.com"
fi

export HOST_DOMAIN
curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/install.sh | bash
`;

module.exports = async function handler(req, res) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { api_id, api_key } = req.body || {};

  if (!api_id || !api_key) {
    return res.status(400).json({ error: 'API ID and API Key are required' });
  }

  const region = 'toronto';
  let scriptId;

  try {
    // 1. Find Ubuntu 24.04 template image (not an ISO)
    const images = await lunaRequest(api_id, api_key, 'image/list', { region: region });
    const imageList = Object.values(images.images || {});
    const ubuntu = imageList.find((img) =>
      img.name && img.name.includes('Ubuntu') && img.name.includes('24.04')
        && img.name.toLowerCase().includes('template')
    );
    if (!ubuntu) {
      return res.status(400).json({ error: 'Ubuntu 24.04 template image not found in region: ' + region + '. Available: ' + imageList.map((i) => i.name).join(', ') });
    }

    // 2. Create startup script
    const scriptRes = await lunaRequest(api_id, api_key, 'script/create', {
      name: 'boltcardhub-init',
      content: STARTUP_SCRIPT,
    });
    scriptId = scriptRes.script_id;

    // 3. Create VM
    const vmRes = await lunaRequest(api_id, api_key, 'vm/create', {
      plan_id: 'm.1s',
      image_id: ubuntu.image_id,
      region: region,
      hostname: 'boltcardhub',
      scripts: scriptId,
    });
    const vmId = vmRes.vm_id;

    // 4. Get VM info for IP (may need to retry while VM is provisioning)
    let vmInfo;
    let ip;
    for (let i = 0; i < 5; i++) {
      vmInfo = await lunaRequest(api_id, api_key, 'vm/info', { vm_id: vmId });
      ip = vmInfo.info && vmInfo.info.ip;
      if (ip) break;
      await new Promise((r) => setTimeout(r, 3000));
    }

    // 5. Derive rDNS hostname from addresses array or fallback
    let hostname = '';
    if (vmInfo.info && vmInfo.info.addresses) {
      const extv4 = vmInfo.info.addresses.find((a) => a.version === '4' && a.external === '1');
      if (extv4 && extv4.reverse) hostname = extv4.reverse.replace(/\.$/, '');
    }
    if (!hostname && ip) {
      hostname = ip.split('.').reverse().join('.') + '.lunanode-rdns.com';
    }

    // 6. Clean up startup script
    try {
      await lunaRequest(api_id, api_key, 'script/delete', { script_id: scriptId });
    } catch {
      // Non-critical — ignore cleanup failure
    }

    return res.status(200).json({
      hostname,
      ip,
      url: `https://${hostname}/admin/`,
    });
  } catch (err) {
    // Clean up script on failure if it was created
    if (scriptId) {
      try {
        await lunaRequest(api_id, api_key, 'script/delete', { script_id: scriptId });
      } catch {
        // Ignore
      }
    }

    return res.status(502).json({ error: err.message || 'Failed to create VM' });
  }
};
