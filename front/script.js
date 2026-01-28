const API_BASE = 'https://localhost:8080'; // при необходимости поменять

async function postAndDownload(url, body, btn, statusEl) {
  try {
    btn.disabled = true;
    statusEl.textContent = 'Загрузка...';

    const resp = await fetch(API_BASE + url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`HTTP ${resp.status}: ${text}`);
    }

    const blob = await resp.blob(); // ожидаем zip
    const disposition = resp.headers.get('Content-Disposition') || '';
    const m = disposition.match(/filename="?([^"]+)"?/i);
    const filename = m ? m[1] : 'files.zip';

    const urlBlob = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = urlBlob;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(urlBlob);

    statusEl.textContent = 'Готово';
  } catch (e) {
    console.error(e);
    statusEl.textContent = 'Ошибка: ' + e.message;
  } finally {
    btn.disabled = false;
  }
}

function toRFC3339DateOnly(d) {
  // сервер ждёт RFC3339, но потом форматирует в "2006-01-02"[file:1]
  // отдаём полночь с локальным часовым поясом
  const dt = new Date(d + 'T00:00:00');
  return dt.toISOString(); // будет с Z, сервер парсит time.RFC3339[file:1]
}

window.addEventListener('DOMContentLoaded', () => {
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
      statusCatalina.textContent = 'Выбери дату';
      return;
    }
    postAndDownload(
      '/api/catalina',
      { timestamp: toRFC3339DateOnly(dateCatalina.value) },
      btnCatalina,
      statusCatalina,
    );
  });

  btnUniverse.addEventListener('click', () => {
    if (!dateUniverse.value) {
      statusUniverse.textContent = 'Выбери дату';
      return;
    }
    postAndDownload(
      '/api/universe',
      { timestamp: toRFC3339DateOnly(dateUniverse.value) },
      btnUniverse,
      statusUniverse,
    );
  });

  btnScaners.addEventListener('click', () => {
    const scanid = scanidInput.value.trim();
    if (!scanid) {
      statusScaners.textContent = 'Укажи ScanID';
      return;
    }
    postAndDownload(
      '/api/scaners',
      { scanid: scanid },
      btnScaners,
      statusScaners,
    );
  });
});
