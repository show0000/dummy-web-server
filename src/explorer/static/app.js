let apis = [];
let selectedAPI = null;
let accessToken = '';
let refreshToken = '';
let jwtEnabled = false;

// --- Init ---
async function init() {
    try {
        const [apisResp, cfgResp] = await Promise.all([
            fetch('/_explorer/apis'),
            fetch('/_explorer/config')
        ]);
        apis = await apisResp.json();
        const cfg = await cfgResp.json();
        jwtEnabled = cfg.jwtEnabled;

        // Hide auth section if JWT is disabled
        if (!jwtEnabled) {
            document.getElementById('auth-section').classList.add('hidden');
        }

        renderAPIList();
    } catch (e) {
        document.getElementById('endpoints').textContent = 'Failed to load APIs.';
    }
}

function renderAPIList() {
    const container = document.getElementById('endpoints');
    container.innerHTML = '';

    if (apis.length === 0) {
        container.textContent = 'No APIs registered.';
        return;
    }

    apis.forEach((api, idx) => {
        const card = document.createElement('div');
        card.className = 'endpoint-card';
        card.dataset.index = idx;
        card.onclick = () => selectAPI(idx);

        let html = `<span class="method-badge method-${api.method}">${api.method}</span>`;
        html += `<span class="endpoint-path">${api.entrypoint}</span>`;
        if (api.description) {
            html += `<div class="endpoint-desc">${api.description}</div>`;
        }
        if (!api.auth) {
            html += `<div class="endpoint-auth">auth: false</div>`;
        }
        card.innerHTML = html;
        container.appendChild(card);
    });
}

function selectAPI(idx) {
    selectedAPI = apis[idx];

    // Highlight active card
    document.querySelectorAll('.endpoint-card').forEach(c => c.classList.remove('active'));
    document.querySelector(`.endpoint-card[data-index="${idx}"]`).classList.add('active');

    // Show test panel
    document.getElementById('test-panel').classList.remove('hidden');
    document.getElementById('response-section').classList.add('hidden');
    document.getElementById('panel-title').textContent = `${selectedAPI.method} ${selectedAPI.entrypoint}`;

    // Path params
    const pathParams = extractPathParams(selectedAPI.entrypoint);
    const paramsSection = document.getElementById('params-section');
    const paramsContainer = document.getElementById('path-params');
    paramsContainer.innerHTML = '';

    if (pathParams.length > 0) {
        paramsSection.classList.remove('hidden');
        pathParams.forEach(p => {
            const row = document.createElement('div');
            row.className = 'kv-row';
            row.innerHTML = `<input type="text" value="${p}" disabled class="pk"><input type="text" placeholder="value" class="pv" data-param="${p}">`;
            paramsContainer.appendChild(row);
        });
    } else {
        paramsSection.classList.add('hidden');
    }

    // Body section: show for POST/PUT/PATCH
    const bodySection = document.getElementById('body-section');
    if (['POST', 'PUT', 'PATCH'].includes(selectedAPI.method)) {
        bodySection.classList.remove('hidden');
    } else {
        bodySection.classList.add('hidden');
    }

    // Reset
    document.getElementById('request-body').value = '';
    document.getElementById('query-params').innerHTML = '<div class="kv-row"><input type="text" placeholder="key" class="qk"><input type="text" placeholder="value" class="qv"></div>';
    document.getElementById('custom-headers').innerHTML = '<div class="kv-row"><input type="text" placeholder="key" class="hk"><input type="text" placeholder="value" class="hv"></div>';
}

function extractPathParams(entrypoint) {
    const matches = entrypoint.match(/\{(\w+)\}/g);
    if (!matches) return [];
    return matches.map(m => m.slice(1, -1));
}

// --- Request ---
async function sendRequest() {
    if (!selectedAPI) return;

    const { url, headers, body } = buildRequest();

    try {
        const opts = { method: selectedAPI.method, headers };
        if (body !== null) opts.body = body;

        const resp = await fetch(url, opts);
        showResponse(resp);
    } catch (e) {
        document.getElementById('response-section').classList.remove('hidden');
        document.getElementById('resp-status').textContent = 'Network Error';
        document.getElementById('resp-status').className = 'status-err';
        document.getElementById('resp-headers').textContent = '';
        document.getElementById('resp-body').textContent = e.message;
    }
}

function buildRequest() {
    let path = selectedAPI.entrypoint;

    // Replace path params
    document.querySelectorAll('#path-params .pv').forEach(input => {
        const param = input.dataset.param;
        path = path.replace(`{${param}}`, encodeURIComponent(input.value));
    });

    // Query params
    const queryParts = [];
    document.querySelectorAll('#query-params .kv-row').forEach(row => {
        const k = row.querySelector('.qk').value.trim();
        const v = row.querySelector('.qv').value;
        if (k) queryParts.push(`${encodeURIComponent(k)}=${encodeURIComponent(v)}`);
    });
    const query = queryParts.length > 0 ? '?' + queryParts.join('&') : '';

    // Headers
    const headers = {};
    document.querySelectorAll('#custom-headers .kv-row').forEach(row => {
        const k = row.querySelector('.hk').value.trim();
        const v = row.querySelector('.hv').value;
        if (k) headers[k] = v;
    });

    // Auto-inject JWT token
    if (accessToken && selectedAPI.auth) {
        headers['Authorization'] = 'Bearer ' + accessToken;
    }

    // Body
    let body = null;
    if (['POST', 'PUT', 'PATCH'].includes(selectedAPI.method)) {
        const bodyText = document.getElementById('request-body').value.trim();
        if (bodyText) {
            body = bodyText;
            if (!headers['Content-Type']) headers['Content-Type'] = 'application/json';
        }
    }

    return { url: path + query, headers, body };
}

async function showResponse(resp) {
    document.getElementById('response-section').classList.remove('hidden');

    const statusEl = document.getElementById('resp-status');
    statusEl.textContent = `${resp.status} ${resp.statusText}`;
    statusEl.className = resp.ok ? 'status-ok' : 'status-err';

    // Headers
    const headerLines = [];
    resp.headers.forEach((v, k) => headerLines.push(`${k}: ${v}`));
    document.getElementById('resp-headers').textContent = headerLines.join('\n');

    // Body
    const text = await resp.text();
    try {
        const json = JSON.parse(text);
        document.getElementById('resp-body').textContent = JSON.stringify(json, null, 2);
    } catch {
        document.getElementById('resp-body').textContent = text;
    }
}

// --- cURL ---
function copyCurl() {
    if (!selectedAPI) return;
    const { url, headers, body } = buildRequest();

    const origin = window.location.origin;
    let cmd = `curl -X ${selectedAPI.method} '${origin}${url}'`;

    for (const [k, v] of Object.entries(headers)) {
        cmd += ` \\\n  -H '${k}: ${v}'`;
    }
    if (body) {
        cmd += ` \\\n  -d '${body}'`;
    }

    navigator.clipboard.writeText(cmd).then(() => {
        const btn = document.getElementById('curl-btn');
        btn.textContent = 'Copied!';
        setTimeout(() => btn.textContent = 'Copy cURL', 1500);
    });
}

// --- Auth ---
function toggleAuthModal() {
    document.getElementById('auth-modal').classList.toggle('hidden');
    document.getElementById('auth-error').textContent = '';
}

async function doLogin() {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const resp = await fetch('/_auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        const data = await resp.json();

        if (!resp.ok) {
            document.getElementById('auth-error').textContent = data.error || 'Login failed';
            return;
        }

        accessToken = data.accessToken;
        refreshToken = data.refreshToken;
        document.getElementById('auth-status').textContent = `Logged in as ${username}`;
        document.getElementById('auth-btn').textContent = 'Auth';
        toggleAuthModal();
    } catch (e) {
        document.getElementById('auth-error').textContent = 'Login request failed';
    }
}

async function doLogout() {
    if (accessToken) {
        try {
            await fetch('/_auth/logout', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + accessToken
                },
                body: JSON.stringify({ refreshToken })
            });
        } catch {}
    }
    accessToken = '';
    refreshToken = '';
    document.getElementById('auth-status').textContent = '';
    document.getElementById('auth-btn').textContent = 'Login';
    toggleAuthModal();
}

// --- Helpers ---
function addQueryParam() {
    const container = document.getElementById('query-params');
    const row = document.createElement('div');
    row.className = 'kv-row';
    row.innerHTML = '<input type="text" placeholder="key" class="qk"><input type="text" placeholder="value" class="qv">';
    container.appendChild(row);
}

function addHeader() {
    const container = document.getElementById('custom-headers');
    const row = document.createElement('div');
    row.className = 'kv-row';
    row.innerHTML = '<input type="text" placeholder="key" class="hk"><input type="text" placeholder="value" class="hv">';
    container.appendChild(row);
}

// Start
init();
