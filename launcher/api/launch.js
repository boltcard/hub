const { lunaRequest } = require('./_luna');

const STARTUP_SCRIPT = `#!/bin/bash
export HOME="\${HOME:-/root}"
export DEBIAN_FRONTEND=noninteractive

# Log all output to file and console
touch /var/log/boltcardhub-install.log
chmod 644 /var/log/boltcardhub-install.log
exec > >(tee -a /var/log/boltcardhub-install.log) 2>&1

# Start a status server on port 8080 so the launcher can read the log via HTTP
python3 -c "
import http.server, socketserver
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        try:
            with open('/var/log/boltcardhub-install.log') as f:
                log = f.read()
        except:
            log = ''
        self.send_response(200)
        self.send_header('Content-Type','text/plain')
        self.end_headers()
        self.wfile.write(log.encode())
    def log_message(self, *a): pass
socketserver.TCPServer(('',8080),H).serve_forever()
" &

echo "[$(date -Iseconds)] Starting Bolt Card Hub install"

# Resolve rDNS hostname from public IP
IP=$(curl -4 -s --retry 5 --retry-delay 2 ifconfig.me)
echo "[$(date -Iseconds)] Public IP: $IP"
HOST_DOMAIN=$(host "$IP" 2>/dev/null | awk '/domain name pointer/ {sub(/[.]$/, "", $NF); print $NF}') || true

if [ -z "$HOST_DOMAIN" ]; then
  IFS='.' read -r a b c d <<< "$IP"
  HOST_DOMAIN="$d.$c.$b.$a.lunanode-rdns.com"
fi

echo "[$(date -Iseconds)] HOST_DOMAIN: $HOST_DOMAIN"
export HOST_DOMAIN
curl -fsSL https://raw.githubusercontent.com/boltcard/hub/main/install.sh | bash
echo "[$(date -Iseconds)] Install script finished"

# Stop the status server
kill %1 2>/dev/null
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
      vm_id: vmId,
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
