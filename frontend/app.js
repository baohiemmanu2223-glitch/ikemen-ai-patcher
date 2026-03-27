// ═══════════════════════════════════════════════════
// IKEMEN AI PATCHER — Frontend Application Logic
// ═══════════════════════════════════════════════════

// State
let state = {
    charPath: '',
    analysis: null,
    styles: [],
    selectedStyle: null,
};

// ─── Navigation ───
document.querySelectorAll('.nav-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        if (btn.disabled) return;
        switchPanel(btn.dataset.panel);
    });
});

function switchPanel(name) {
    document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
    document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('panel-' + name).classList.add('active');
    document.querySelector(`[data-panel="${name}"]`).classList.add('active');
}

function enableNav(name) {
    document.querySelector(`[data-panel="${name}"]`).disabled = false;
}

// ─── Loading Overlay ───
function showLoading(text) {
    document.getElementById('loading-text').textContent = text || 'Processing...';
    document.getElementById('loading-overlay').classList.remove('hidden');
}

function hideLoading() {
    document.getElementById('loading-overlay').classList.add('hidden');
}

// ─── API 1: Select Character ───
async function selectCharacter() {
    try {
        const path = await window.go.main.App.SelectCharacter();
        if (!path) return;
        state.charPath = path;
        document.getElementById('path-display').textContent = path;
        document.getElementById('selected-path').classList.remove('hidden');
        document.getElementById('btn-analyze').classList.remove('hidden');
    } catch (e) {
        console.error('Select error:', e);
    }
}

// ─── API 2: Analyze Character ───
async function analyzeCharacter() {
    if (!state.charPath) return;
    showLoading('Analyzing character...');
    document.getElementById('btn-analyze').disabled = true;

    try {
        const json = await window.go.main.App.AnalyzeCharacter(state.charPath);
        const data = JSON.parse(json);
        state.analysis = data;

        if (!data.success) {
            hideLoading();
            alert('Analysis failed: ' + data.error);
            document.getElementById('btn-analyze').disabled = false;
            return;
        }

        renderAnalysis(data);
        enableNav('analysis');
        enableNav('styles');
        enableNav('patch');
        enableNav('report');
        switchPanel('analysis');

        // Auto-load styles
        loadStyles();
    } catch (e) {
        console.error('Analyze error:', e);
        alert('Analysis error: ' + e);
    } finally {
        hideLoading();
        document.getElementById('btn-analyze').disabled = false;
    }
}

function renderAnalysis(data) {
    document.getElementById('char-name').textContent = data.charName || 'Unknown';
    document.getElementById('total-states').textContent = data.totalStates + ' states';

    // Stats
    document.getElementById('stat-normals').textContent = (data.normals || []).length;
    document.getElementById('stat-specials').textContent = (data.specials || []).length;
    document.getElementById('stat-hypers').textContent = (data.hypers || []).length;
    document.getElementById('stat-chains').textContent = (data.comboChains || []).length;
    
    // Variables
    document.getElementById('stat-unused-vars').textContent = (data.unusedVars || []).join(', ') || 'None';
    document.getElementById('stat-unused-fvars').textContent = (data.unusedFvars || []).join(', ') || 'None';
    document.getElementById('stat-unused-sysvars').textContent = (data.unusedSysvars || []).join(', ') || 'None';
    
    const usageContainer = document.getElementById('var-usages');
    usageContainer.innerHTML = '';
    (data.varUsages || []).forEach(u => {
        const div = document.createElement('div');
        div.style.marginBottom = '8px';
        div.style.padding = '5px';
        div.style.background = 'rgba(255,255,255,0.05)';
        div.style.borderRadius = '4px';
        
        let writeStates = u.SetIn && u.SetIn.length > 0 ? u.SetIn.join(', ') : 'None';
        let readStates = u.UsedIn && u.UsedIn.length > 0 ? u.UsedIn.join(', ') : 'None';
        
        div.innerHTML = `<strong>${u.Type}(${u.Index})</strong><br>
        <span style="color:#ff6b6b">Set In: [${writeStates}]</span><br>
        <span style="color:#4dabf7">Read In: [${readStates}]</span>`;
        usageContainer.appendChild(div);
    });

    // State IDs
    document.getElementById('normal-ids').textContent =
        (data.normals || []).join(', ') || 'None';

    // Combos
    const chainList = document.getElementById('chain-list');
    chainList.innerHTML = '';
    (data.comboChains || []).forEach((c, i) => {
        const li = document.createElement('li');
        li.textContent = `${i + 1}. ${c.description}`;
        chainList.appendChild(li);
    });

    // Risk
    document.getElementById('stat-risks').textContent = (data.riskStates || []).length;
    document.getElementById('stat-valid').textContent =
        data.validation.valid ? '✅ PASS' : '❌ FAIL';
    document.getElementById('stat-valid').style.color =
        data.validation.valid ? 'var(--accent-green)' : 'var(--accent-red)';

    const riskList = document.getElementById('risk-list');
    riskList.innerHTML = '';
    (data.riskStates || []).slice(0, 15).forEach(r => {
        const li = document.createElement('li');
        li.className = 'risk-' + r.severity;
        li.textContent = `[${r.severity}] State ${r.stateId}: ${r.reason}`;
        riskList.appendChild(li);
    });
}

// ─── API 3: Load Styles ───
async function loadStyles() {
    try {
        const json = await window.go.main.App.LoadStyles();
        state.styles = JSON.parse(json) || [];
        renderStyles();
    } catch (e) {
        console.error('Load styles error:', e);
    }
}

function renderStyles() {
    const grid = document.getElementById('styles-grid');
    grid.innerHTML = '';

    if (state.styles.length === 0) {
        grid.innerHTML = '<div class="glass-card"><p style="color:var(--text-secondary)">No .zss styles found in styles/ directory</p></div>';
        return;
    }

    // Auto-generated option
    const autoCard = document.createElement('div');
    autoCard.className = 'glass-card style-card' + (!state.selectedStyle ? ' selected' : '');
    autoCard.innerHTML = `
        <h4>🤖 Auto-Generated</h4>
        <p class="style-desc">Default boss-level AI based on character analysis</p>
        <div class="style-meta">
            <span class="style-tag">adaptive</span>
            <span class="style-tag">3-layer</span>
        </div>
    `;
    autoCard.onclick = () => selectStyle(null);
    grid.appendChild(autoCard);

    state.styles.forEach(s => {
        const card = document.createElement('div');
        card.className = 'glass-card style-card' +
            (state.selectedStyle && state.selectedStyle.name === s.name ? ' selected' : '');
        card.innerHTML = `
            <h4>🎨 ${s.name}</h4>
            <p class="style-desc">${s.description || 'Custom AI style'}</p>
            <div class="style-meta">
                <span class="style-tag">${s.decisions} decisions</span>
                ${s.reactionDelay ? `<span class="style-tag">delay:${s.reactionDelay}t</span>` : ''}
                ${s.hasAdaptive ? '<span class="style-tag adaptive">adaptive</span>' : ''}
            </div>
        `;
        card.onclick = () => selectStyle(s);
        grid.appendChild(card);
    });
}

async function selectStyle(s) {
    state.selectedStyle = s;
    renderStyles();

    // Update patch panel
    const el = document.getElementById('current-style');
    if (s) {
        el.innerHTML = `
            <span class="style-badge">${s.name}</span>
            <p>${s.description || 'Custom style'} — ${s.decisions} decisions</p>
        `;
    } else {
        el.innerHTML = `
            <span class="style-badge">Auto-Generated</span>
            <p>Default boss-level AI will be auto-generated based on character analysis</p>
        `;
    }

    // Preview
    const preview = document.getElementById('style-preview');
    if (s && s.filePath) {
        try {
            const json = await window.go.main.App.PreviewStyle(s.filePath);
            const info = JSON.parse(json);
            document.getElementById('style-preview-content').textContent = info.summary || '';
            preview.classList.remove('hidden');
        } catch (e) {
            preview.classList.add('hidden');
        }
    } else {
        preview.classList.add('hidden');
    }
}

// ─── API 4: Apply Patch ───
async function applyPatch() {
    if (!state.charPath) return;
    const btn = document.getElementById('btn-patch');
    const status = document.getElementById('patch-status');
    const result = document.getElementById('patch-result');

    btn.disabled = true;
    status.textContent = '⏳ Generating and patching AI...';
    status.className = 'patch-status loading';
    status.classList.remove('hidden');
    result.classList.add('hidden');

    const config = {
        stylePath: state.selectedStyle ? state.selectedStyle.filePath : '',
        antiSpam: document.getElementById('opt-antispam').checked,
        bossMode: document.getElementById('opt-boss').checked,
        randomness: document.getElementById('opt-randomness').value / 100,
    };

    try {
        const json = await window.go.main.App.ApplyPatch(state.charPath, JSON.stringify(config));
        const data = JSON.parse(json);

        status.classList.add('hidden');

        if (data.success) {
            result.className = 'patch-result success';
            result.innerHTML = `
                <strong>✅ Patch Successful!</strong><br>
                Style: ${data.styleUsed}<br>
                Patched: ${(data.patchedFiles || []).join(', ')}<br>
                Backup: ${(data.backupFiles || []).join(', ')}<br>
                ${data.reportPath ? `Report: ${data.reportPath}` : ''}
            `;
        } else {
            result.className = 'patch-result error';
            result.innerHTML = `<strong>❌ Patch Failed</strong><br>${data.error}`;
        }
        result.classList.remove('hidden');
    } catch (e) {
        status.classList.add('hidden');
        result.className = 'patch-result error';
        result.innerHTML = `<strong>❌ Error</strong><br>${e}`;
        result.classList.remove('hidden');
    } finally {
        btn.disabled = false;
    }
}

// ─── API 5: Generate Report ───
async function generateReport() {
    if (!state.charPath) return;
    showLoading('Generating report...');

    try {
        const content = await window.go.main.App.GenerateReport(state.charPath);
        document.getElementById('report-content').textContent = content;
    } catch (e) {
        document.getElementById('report-content').textContent = 'Error: ' + e;
    } finally {
        hideLoading();
    }
}

// ─── UI Helpers ───
function updateSlider() {
    const val = document.getElementById('opt-randomness').value;
    document.getElementById('rand-value').textContent = val + '%';
}

// ─── API 6: Load About ───
async function loadAbout() {
    try {
        const res = await fetch('about.json');
        if (res.ok) {
            const data = await res.json();
            const container = document.getElementById('about-container');
            container.innerHTML = `
                <h2 style="margin-bottom: 20px; color: var(--text-primary); text-align: center;">ℹ️ About Ikemen AI Patcher</h2>
                <div class="stat-row" style="margin-bottom: 15px;">
                    <span class="stat-label">Author</span>
                    <span class="stat-value" style="color: var(--accent-blue)">${data.Author}</span>
                </div>
                <div class="stat-row" style="margin-bottom: 15px;">
                    <span class="stat-label">Version</span>
                    <span class="stat-value">${data.Version}</span>
                </div>
                <div class="stat-row" style="margin-bottom: 15px;">
                    <span class="stat-label">Contact</span>
                    <span class="stat-value">${data.Contact}</span>
                </div>
                <div class="stat-row" style="margin-bottom: 15px;">
                    <span class="stat-label">Github link</span>
                    <span class="stat-value">${data['Github link'] || '—'}</span>
                </div>
                <div class="stat-row" style="margin-bottom: 20px;">
                    <span class="stat-label">Donate</span>
                    <span class="stat-value" style="color: var(--accent-green)">${data.Donate}</span>
                </div>
                ${data.QR ? `<div style="text-align: center;"><img src="${data.QR}" alt="Donate QR" style="max-width: 250px; border-radius: 8px; border: 2px solid var(--border-color);"></div>` : ''}
            `;
        }
    } catch (e) {
        console.error("Failed to load about.json", e);
    }
}

// ─── Initialization ───
document.addEventListener('DOMContentLoaded', () => {
    loadAbout();
});
