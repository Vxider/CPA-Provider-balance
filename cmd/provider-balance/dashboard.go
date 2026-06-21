package main

// renderDashboardHTML builds the provider-balance dashboard page. The page
// loads once on open and refreshes via the manual button.
func renderDashboardHTML() string {
	return dashboardHTML
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="icon" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjNGZlM2M1IiBzdHJva2Utd2lkdGg9IjIiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCI+PHBhdGggZD0ibTE2IDE2IDMtOCAzIDhjLS44Ny42NS0xLjkyIDEtMyAxcy0yLjEzLS4zNS0zLTFaIi8+PHBhdGggZD0ibTIgMTYgMy04IDMgOGMtLjg3LjY1LTEuOTIgMS0zIDFzLTIuMTMtLjM1LTMtMVoiLz48cGF0aCBkPSJNNyAyMWgxMCIvPjxwYXRoIGQ9Ik0xMiAzdjE4Ii8+PHBhdGggZD0iTTMgN2gyYzIgMCA1LTEgNy0yIDIgMSA1IDIgNyAyaDIiLz48L3N2Zz4=">
<title>Provider Balance</title>
<style>
  :root {
    --bg-0: #0b1020;
    --bg-1: #111933;
    --panel: rgba(20, 28, 56, 0.72);
    --panel-border: rgba(122, 145, 210, 0.18);
    --ink: #e8ecf8;
    --ink-dim: #9aa4c4;
    --ink-faint: #5b6486;
    --accent: #4fe3c5;
    --accent-2: #7aa0ff;
    --warn: #ffd166;
    --bad: #ff6b8a;
    --good: #5fe3a6;
  }
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }
  body {
    font-family: "Sora", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
    color: var(--ink);
    min-height: 100vh;
    background:
      radial-gradient(1200px 600px at 85% -10%, rgba(79,227,197,0.15), transparent 60%),
      radial-gradient(900px 500px at -10% 110%, rgba(122,160,255,0.14), transparent 60%),
      linear-gradient(160deg, var(--bg-0), var(--bg-1) 70%, #0a0e22);
    padding: 28px clamp(16px, 4vw, 56px) 64px;
  }
  .mono { font-family: "JetBrains Mono", "SF Mono", ui-monospace, monospace; }
  header.top {
    display: flex; align-items: baseline; gap: 18px; flex-wrap: wrap;
    margin-bottom: 24px;
  }
  header.top h1 {
    font-size: clamp(22px, 3.4vw, 34px);
    font-weight: 700; margin: 0; letter-spacing: -0.5px;
    display: inline-flex; align-items: center; gap: 10px;
    background: linear-gradient(92deg, var(--accent), var(--accent-2));
    -webkit-background-clip: text; background-clip: text; color: transparent;
  }
  .logo { width: 28px; height: 28px; color: #4fe3c5; flex: none; }
  .refresh-btn {
    margin-left: auto; padding: 6px 12px; border-radius: 999px;
    font-size: 12px; line-height: 1; color: var(--ink-dim);
    height: 28px; box-sizing: border-box; align-self: center;
    background: transparent; border: 1px solid transparent;
    display: inline-flex; align-items: center; gap: 8px; cursor: pointer;
    transition: color .2s ease, border-color .2s ease;
  }
  .refresh-btn:hover, .refresh-btn:focus-visible { color: var(--ink); outline: none; }
  .refresh-btn:focus { outline: none; }
  .refresh-btn svg { width: 14px; height: 14px; }
  .refresh-btn.loading svg { animation: spin .8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  header.top .sub { color: var(--ink-dim); font-size: 14px; }
  .status-pill {
    padding: 6px 12px; border-radius: 999px;
    font-size: 12px; line-height: 1; color: var(--ink-dim);
    height: 28px; box-sizing: border-box; align-self: center;
    background: rgba(255,255,255,0.04); border: 1px solid var(--panel-border);
    display: inline-flex; align-items: center; gap: 8px;
  }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--ink-faint); }
  .dot.live { background: var(--good); box-shadow: 0 0 10px var(--good); animation: pulse 1.6s infinite; }
  .dot.err  { background: var(--bad); }
  @keyframes pulse { 0%,100% {opacity:1} 50% {opacity:0.4} }
  .summary {
    display: grid; gap: 14px; margin-bottom: 26px;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  }
  .summary .card {
    padding: 16px 18px; border-radius: 14px;
    background: var(--panel); border: 1px solid var(--panel-border);
    backdrop-filter: blur(8px);
  }
  .summary .card .label { font-size: 12px; color: var(--ink-faint); text-transform: uppercase; letter-spacing: 1px; }
  .summary .card .value { font-size: 26px; font-weight: 700; margin-top: 4px; }
  .summary .card .value.dim { color: var(--ink-dim); font-weight: 500; }
  .table-wrap {
    border-radius: 14px; overflow-x: auto;
    background: var(--panel); border: 1px solid var(--panel-border);
    backdrop-filter: blur(8px);
  }
  table.grid {
    width: 100%; border-collapse: separate; border-spacing: 0; min-width: max-content;
  }
  table.grid th, table.grid td {
    padding: 12px 16px; text-align: left; font-size: 14px;
    border-bottom: 1px solid rgba(122,145,210,0.10);
  }
  table.grid th {
    font-size: 11px; text-transform: uppercase; letter-spacing: 1.2px;
    color: var(--ink-faint); font-weight: 600;
    background: rgba(0,0,0,0.18);
  }
  table.grid tr:last-child td { border-bottom: none; }
  table.grid tr.row-ok td .num { color: var(--good); }
  td .kind { color: var(--accent-2); font-size: 12px; }
  td .note { color: var(--ink-faint); font-size: 12px; max-width: 340px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  td.provider { white-space: nowrap; }
  td.status { font-weight: 600; white-space: nowrap; }
  td.status-ok { color: var(--good); }
  td.status-err { color: var(--bad); }
  td.status-na { color: var(--ink-dim); }
  .bar { height: 6px; border-radius: 3px; background: rgba(255,255,255,0.08); overflow: hidden; min-width: 90px; margin-top: 4px; }
  .bar > span { display: block; height: 100%; border-radius: 3px; background: linear-gradient(90deg, var(--accent), var(--accent-2)); }
  .bar.high > span { background: linear-gradient(90deg, var(--good), var(--accent)); }
  .bar.low  > span { background: linear-gradient(90deg, var(--warn), var(--bad)); }
  tbody tr { animation: fadein .5s ease both; }
  tbody tr:nth-child(1){animation-delay:.02s} tbody tr:nth-child(2){animation-delay:.05s}
  tbody tr:nth-child(3){animation-delay:.08s} tbody tr:nth-child(4){animation-delay:.11s}
  tbody tr:nth-child(5){animation-delay:.14s} tbody tr:nth-child(6){animation-delay:.17s}
  tbody tr:nth-child(7){animation-delay:.20s} tbody tr:nth-child(8){animation-delay:.23s}
  @keyframes fadein { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: none; } }
  .empty { padding: 40px; text-align: center; color: var(--ink-faint); }
  footer { margin-top: 22px; color: var(--ink-faint); font-size: 12px; text-align: center; }
  @media (max-width: 640px) {
    .hide-sm { display: none; }
    table.grid th, table.grid td { padding: 10px 12px; font-size: 13px; }
  }
</style>
</head>
<body>
  <header class="top">
    <h1><svg class="logo" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m16 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z"/><path d="m2 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z"/><path d="M7 21h10"/><path d="M12 3v18"/><path d="M3 7h2c2 0 5-1 7-2 2 1 5 2 7 2h2"/></svg>Provider Balance</h1>
    <span class="sub" id="when">--</span>
    <button class="refresh-btn" id="refreshBtn" type="button" title="刷新"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-2.64-6.36"/><path d="M21 3v6h-6"/></svg></button>
    <span class="status-pill"><span class="dot" id="dot"></span><span id="pill">idle</span></span>
  </header>

  <section class="summary" id="summary"></section>

  <div class="table-wrap">
  <table class="grid">
    <thead>
      <tr>
        <th>Provider</th>
        <th class="hide-sm">Base URL</th>
        <th>Remaining</th>
        <th class="hide-sm">Used</th>
        <th class="hide-sm">Total</th>
        <th>Status</th>
        <th class="hide-sm">Note</th>
      </tr>
    </thead>
    <tbody id="rows">
      <tr><td colspan="7" class="empty">Loading...</td></tr>
    </tbody>
  </table>
  </div>

  <footer>provider-balance plugin</footer>

<script>
const API = "/v0/resource/plugins/provider-balance/balance.json";
const $ = (id) => document.getElementById(id);

function fmtNum(v, unit) {
  if (v === null || v === undefined) return "-";
  const n = Number(v);
  if (!isFinite(n)) return "-";
  let s = n >= 100 ? n.toFixed(0) : n.toFixed(2);
  return s + (unit && unit !== "-" ? " " + unit : "");
}

function pct(remaining, total) {
  if (remaining == null || total == null || total === 0) return null;
  let p = (remaining / total) * 100;
  if (p < 0) p = 0; if (p > 100) p = 100;
  return p;
}

async function load() {
  $("dot").className = "dot live";
  $("pill").textContent = "fetching";
  const btn = $("refreshBtn");
  btn.classList.add("loading"); btn.disabled = true;
  try {
    const res = await fetch(API, { headers: { "Accept": "application/json" } });
    if (!res.ok) throw new Error("HTTP " + res.status);
    const data = await res.json();
    render(data);
    $("dot").className = "dot live";
    $("pill").textContent = "ok";
  } catch (e) {
    $("dot").className = "dot err";
    $("pill").textContent = "error";
    $("rows").innerHTML = '<tr><td colspan="7" class="empty">Failed to load: ' + String(e.message || e) + '</td></tr>';
    $("summary").innerHTML = "";
  } finally {
    btn.classList.remove("loading"); btn.disabled = false;
  }
  $("when").textContent = "updated " + new Date().toLocaleTimeString();
}

function render(data) {
  const rows = (data.providers || []).slice().sort((a,b) => (a.provider||"").localeCompare(b.provider||""));
  const okCount = rows.filter(r => r.status === "OK").length;
  const errCount = rows.filter(r => r.status === "Err").length;
  let remSum = 0, remHas = 0;
  rows.forEach(r => {
    if (typeof r.remaining === "number") { remSum += r.remaining; remHas++; }
  });
  $("summary").innerHTML = summaryCards(rows.length, okCount, errCount, remSum, remHas);
  if (!rows.length) {
    $("rows").innerHTML = '<tr><td colspan="7" class="empty">No providers configured. Set openai-compatibility / codex-api-key in config.yaml or extra_providers in the plugin config.</td></tr>';
    return;
  }
  $("rows").innerHTML = rows.map(r => {
    const cls = r.status === "OK" ? "row-ok" : "";
    const p = pct(r.remaining, r.total);
    let barCls = "bar", barW = "0%";
    if (p != null) { barW = p.toFixed(1) + "%"; barCls += p > 50 ? " high" : " low"; }
    const bar = '<div class="' + barCls + '"><span style="width:' + (p!=null?barW:"0%") + '"></span></div>';
    return '<tr class="' + cls + '">'
      + '<td class="provider"><div>' + esc(r.provider || "-") + '</div><div class="kind mono">' + esc(r.kind || "") + '</div></td>'
      + '<td class="hide-sm mono" title="' + esc(r.base_url||"") + '">' + esc(shortUrl(r.base_url)) + '</td>'
      + '<td class="num mono">' + fmtNum(r.remaining, r.unit) + bar + '</td>'
      + '<td class="hide-sm mono">' + fmtNum(r.used, "") + '</td>'
      + '<td class="hide-sm mono">' + fmtNum(r.total, "") + '</td>'
      + '<td class="status ' + (r.status === "OK" ? "status-ok" : r.status === "Err" ? "status-err" : "status-na") + '">' + esc(r.status || "-") + '</td>'
      + '<td class="hide-sm"><div class="note" title="' + esc(r.note||"") + '">' + esc(r.note || "-") + '</div></td>'
      + '</tr>';
  }).join("");
}

function summaryCards(total, ok, errN, remSum, remHas) {
  const remCard = remHas > 0
    ? '<div class="value">' + (remSum >= 100 ? remSum.toFixed(0) : remSum.toFixed(2)) + '</div>'
    : '<div class="value dim">--</div>';
  return ''
    + '<div class="card"><div class="label">Providers</div><div class="value">' + total + '</div></div>'
    + '<div class="card"><div class="label">Healthy</div><div class="value">' + ok + '</div></div>'
    + '<div class="card"><div class="label">Errors</div><div class="value ' + (errN? "" : "dim") + '">' + errN + '</div></div>'
    + '<div class="card"><div class="label">Summed Remaining</div>' + remCard + '</div>';
}

function esc(s) { return String(s==null?"":s).replace(/[&<>"]/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;"}[c])); }
function shortUrl(u) {
  if (!u) return "-";
  try { let x = new URL(u); return x.host + (x.pathname && x.pathname !== "/" ? x.pathname : ""); }
  catch { return u.length > 40 ? u.slice(0,40)+"..." : u; }
}

$("refreshBtn").addEventListener("click", load);
load();
</script>
</body>
</html>`
