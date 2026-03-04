const btn = document.getElementById('scanButton');
const results = document.getElementById('results');
const noNfc = document.getElementById('no-nfc');
const hasNfc = 'NDEFReader' in window;

if (hasNfc) {
    btn.disabled = false;
} else {
    btn.textContent = 'NFC Not Supported';
    noNfc.style.display = 'block';
}

const MONTHS = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];

function formatDate(ts) {
    const d = new Date(ts * 1000);
    return d.getDate() + ' ' + MONTHS[d.getMonth()] + ' ' + d.getFullYear();
}

function formatTime(ts) {
    return new Date(ts * 1000).toLocaleTimeString();
}

function el(tag, cls, text) {
    const e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
}

function render(data) {
    results.replaceChildren();

    results.appendChild(el('div', 'balance', data.AvailableBalance.toLocaleString() + ' sats'));
    results.appendChild(el('div', 'balance-label', 'Available Balance'));

    if (data.txs && data.txs.length > 0) {
        const table = el('table');
        const thead = el('thead');
        const hr = el('tr');
        hr.appendChild(el('th', null, 'Date'));
        hr.appendChild(el('th', null, 'Time'));
        hr.appendChild(el('th', 'num', 'Amount'));
        hr.appendChild(el('th', 'num', 'Fees'));
        thead.appendChild(hr);
        table.appendChild(thead);

        const tbody = el('tbody');
        for (const tx of data.txs) {
            const tr = el('tr');
            tr.appendChild(el('td', null, formatDate(tx.Timestamp)));
            tr.appendChild(el('td', null, formatTime(tx.Timestamp)));
            const amtCls = 'num' + (tx.AmountSats >= 0 ? ' positive' : '');
            tr.appendChild(el('td', amtCls, tx.AmountSats.toLocaleString()));
            tr.appendChild(el('td', 'num', tx.FeeSats.toLocaleString()));
            tbody.appendChild(tr);
        }
        table.appendChild(tbody);
        results.appendChild(table);
    }

    results.style.display = 'block';
}

btn.addEventListener('click', async () => {
    btn.textContent = 'Scanning\u2026';
    btn.disabled = true;
    results.style.display = 'none';

    try {
        const reader = new NDEFReader();
        await reader.scan();

        reader.addEventListener('reading', ({ message }) => {
            if (message.records.length === 0) return;

            const url = new TextDecoder('utf-8').decode(message.records[0].data);

            fetch('/balance-ajax?card=' + encodeURIComponent(url))
                .then(r => r.json())
                .then(data => {
                    render(data);
                    btn.textContent = 'Scan Again';
                    btn.disabled = false;
                })
                .catch(() => {
                    btn.textContent = 'Scan Card';
                    btn.disabled = false;
                });
        });
    } catch {
        btn.textContent = 'Scan Card';
        btn.disabled = false;
    }
});
