
const NODES_TOMCAT = {
  node1:                 'https://node1:8080',
  node2:                 'https://node2:8080',
};
const SCANERS_API_BASE = 'https://node3:8080'; 

function setStatus(statusEl, type, message) {
  statusEl.classList.remove('status--loading', 'status--success', 'status--error');
  statusEl.innerHTML = '';

  if (type) {
    statusEl.classList.add(`status--${type}`);
  }

  if (type === 'loading') {
    const spinner = document.createElement('span');
    spinner.className = 'spinner';
    statusEl.appendChild(spinner);
  }

  if (message) {
    const text = document.createElement('span');
    text.textContent = message;
    statusEl.appendChild(text);
  }
}

async function postAndDownloadMultiple(nodes, endpoint, body, btn, statusEl) {
  const controller = new AbortController();
  const timeoutMs = 600000;
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    btn.disabled = true;
    setStatus(statusEl, 'loading', 'Формирование архивов...');

    const promises = nodes.map(async (node) => {
      const base = NODES_TOMCAT[node];
      if (!base) {
        throw new Error(`Неизвестный node: ${node}`);
      }

      const resp = await fetch(base + endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!resp.ok) {
        const text = await resp.text().catch(() => '');
        throw new Error(`${node}: HTTP ${resp.status}${text ? `: ${text}` : ''}`);
      }

      const blob = await resp.blob();
      const disposition = resp.headers.get('Content-Disposition') || '';
      const m = disposition.match(/filename=\"?([^\"]+)\"?/i);
      const baseName = m ? m[1].replace('.zip', '') : 'files';
      const filename = `${node}-${baseName}.zip`;

      const urlBlob = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = urlBlob;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(urlBlob);

      return { node, filename };
    });

    const results = await Promise.allSettled(promises);
    const errors = results.filter(r => r.status === 'rejected');

    if (errors.length > 0) {
      const errorMsgs = errors.map(r => r.reason.message).join('; ');
      throw new Error(errorMsgs);
    }

    setStatus(
      statusEl,
      'success',
      `Архивы скачаны: ${results.map(r => r.value.filename).join(', ')}`
    );

  } catch (e) {
    if (e.name === 'AbortError') {
      setStatus(statusEl, 'error', 'Таймаут ожидания ответа');
    } else {
      console.error(e);
      setStatus(statusEl, 'error', `Ошибка: ${e.message}`);
    }
  } finally {
    clearTimeout(timeoutId);
    btn.disabled = false;
  }
}

async function postAndDownload(url, body, btn, statusEl) {
  const controller = new AbortController();
  const timeoutMs = 600000;
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    btn.disabled = true;
    setStatus(statusEl, 'loading', 'Формирование архива...');

    const resp = await fetch(SCANERS_API_BASE + url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal: controller.signal,
    });

    if (!resp.ok) {
      const text = await resp.text().catch(() => '');
      throw new Error(`HTTP ${resp.status}${text ? `: ${text}` : ''}`);
    }

    const blob = await resp.blob();
    const disposition = resp.headers.get('Content-Disposition') || '';
    const m = disposition.match(/filename=\"?([^\"]+)\"?/i);
    const filename = m ? m[1] : 'files.zip';

    const urlBlob = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = urlBlob;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(urlBlob);

    setStatus(statusEl, 'success', 'Архив скачан');
  } catch (e) {
    if (e.name === 'AbortError') {
      setStatus(statusEl, 'error', 'Таймаут ожидания ответа');
    } else {
      console.error(e);
      setStatus(statusEl, 'error', `Ошибка: ${e.message}`);
    }
  } finally {
    clearTimeout(timeoutId);
    btn.disabled = false;
  }
}


function toRFC3339DateOnly(d) {
  // Парсим YYYY-MM-DD и создаём дату в UTC без смещения
  const [year, month, day] = d.split('-').map(Number);
  const date = Date.UTC(year, month - 1, day); // month -1 т.к. JS месяцы 0-based
  const dt = new Date(date);
  return dt.toISOString();
}


window.addEventListener('DOMContentLoaded', () => {
  const app = document.querySelector('.app') || document.body;

  const btnCatalina = document.getElementById('btn-catalina');
  const dateCatalina = document.getElementById('date-catalina');
  const statusCatalina = document.getElementById('status-catalina');

  const btnUniverse = document.getElementById('btn-universe');
  const dateUniverse = document.getElementById('date-universe');
  const statusUniverse = document.getElementById('status-universe');

  const btnScaners = document.getElementById('btn-scaners');
  const scanidInput = document.getElementById('scanid');
  const statusScaners = document.getElementById('status-scaners');

  btnCatalina.addEventListener('click', () => {
  if (!dateCatalina.value) {
    setStatus(statusCatalina, 'error', 'Выберите дату');
    return;
  }

  postAndDownloadMultiple(
    ['node1', 'node2'],
    '/api/catalina',
    { timestamp: toRFC3339DateOnly(dateCatalina.value) },
    btnCatalina,
    statusCatalina
  );
});

btnUniverse.addEventListener('click', () => {
  if (!dateUniverse.value) {
    setStatus(statusUniverse, 'error', 'Выберите дату');
    return;
  }

  postAndDownloadMultiple(
    ['node1', 'node2'],
    '/api/universe',
    { timestamp: toRFC3339DateOnly(dateUniverse.value) },
    btnUniverse,
    statusUniverse
  );
});

btnScaners.addEventListener('click', () => {
  const scanid = scanidInput.value.trim();
  if (!scanid) {
    setStatus(statusScaners, 'error', 'Укажите ScanID');
    return;
  }

  postAndDownload(
    '/api/scaners',
    { scanid: scanid },
    btnScaners,
    statusScaners
  );
});

});
