// Tests de la caché de api.js. Lo que se protege aquí es la regla de negocio que
// más duele si se rompe: tras crear un usuario o una suscripción, el siguiente
// listado tiene que venir del servidor, no de la caché.
//
// Ejecutar con:  npx vitest run src/utils/api.test.js

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';

let api, invalidate, clearCache, getJSON;

function jsonResponse(payload, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

beforeEach(async () => {
  vi.resetModules();
  // api.js deriva la URL base de window.location; se stubea en lugar de añadir
  // jsdom, que no aporta nada más para estos tests.
  vi.stubGlobal('window', {
    location: { protocol: 'http:', host: 'localhost:8080', href: '/' },
  });
  vi.stubGlobal('localStorage', {
    store: {},
    getItem(k) { return this.store[k] ?? null; },
    setItem(k, v) { this.store[k] = v; },
    removeItem(k) { delete this.store[k]; },
  });
  vi.stubGlobal('fetch', vi.fn());
  ({ api, invalidate, clearCache, getJSON } = await import('./api.js'));
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('caché de GET', () => {
  it('sirve la segunda lectura desde la caché', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [{ id: 1 }] }));

    const first = await api.get('/users');
    const second = await api.get('/users');

    expect(fetch).toHaveBeenCalledTimes(1);
    // Ambas respuestas deben tener cuerpo legible e idéntico.
    expect(await first.json()).toEqual({ data: [{ id: 1 }] });
    expect(await second.json()).toEqual({ data: [{ id: 1 }] });
  });

  it('no cachea sub-recursos, solo listados', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: { id: 'abc' } }));

    await api.get('/users/abc/profile');
    await api.get('/users/abc/profile');
    await api.get('/users/by-document?document=123');
    await api.get('/users/by-document?document=123');

    expect(fetch).toHaveBeenCalledTimes(4);
  });

  it('respeta noCache', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));

    await api.get('/users', { noCache: true });
    await api.get('/users', { noCache: true });

    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('distingue querystrings distintos', async () => {
    fetch.mockResolvedValue(jsonResponse([]));

    await api.get('/subscriptions?status=ACTIVE');
    await api.get('/subscriptions?status=EXPIRED');
    await api.get('/subscriptions?status=ACTIVE');

    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('no cachea respuestas de error', async () => {
    fetch.mockResolvedValue(jsonResponse({ error: 'boom' }, 500));

    await api.get('/users');
    await api.get('/users');

    expect(fetch).toHaveBeenCalledTimes(2);
  });
});

describe('deduplicación de peticiones en vuelo', () => {
  it('colapsa lecturas concurrentes en una sola petición', async () => {
    let resolveFetch;
    fetch.mockReturnValue(new Promise(resolve => { resolveFetch = resolve; }));

    const all = Promise.all([api.get('/users'), api.get('/users'), api.get('/users')]);
    resolveFetch(jsonResponse({ data: [{ id: 1 }] }));
    const results = await all;

    expect(fetch).toHaveBeenCalledTimes(1);
    // Cada llamador recibe su propio clon legible.
    for (const res of results) {
      expect(await res.json()).toEqual({ data: [{ id: 1 }] });
    }
  });
});

describe('invalidación por mutación', () => {
  it('crear un usuario obliga a releer el listado', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));
    await api.get('/users');
    expect(fetch).toHaveBeenCalledTimes(1);

    await api.post('/users', { first_name: 'Ana' });
    await api.get('/users');

    // 1 GET inicial + 1 POST + 1 GET fresco
    expect(fetch).toHaveBeenCalledTimes(3);
  });

  it('crear una suscripción obliga a releer suscripciones', async () => {
    fetch.mockResolvedValue(jsonResponse([]));
    await api.get('/subscriptions?status=ACTIVE');
    await api.post('/subscriptions', { user_id: 'x' });
    await api.get('/subscriptions?status=ACTIVE');

    expect(fetch).toHaveBeenCalledTimes(3);
  });

  it('mutar por id invalida el listado de la raíz', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));
    await api.get('/users');
    await api.put('/users/abc', { first_name: 'B' });
    await api.get('/users');

    expect(fetch).toHaveBeenCalledTimes(3);
  });

  it('una venta invalida productos porque descuenta stock', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));
    await api.get('/products');
    await api.post('/sales', { details: [] });
    await api.get('/products');

    expect(fetch).toHaveBeenCalledTimes(3);
  });

  it('la invalidación no arrastra recursos con nombre parecido', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));
    await api.get('/plans');
    await api.get('/payment-methods');

    invalidate('/plans');

    await api.get('/plans');          // relee
    await api.get('/payment-methods'); // sigue en caché
    expect(fetch).toHaveBeenCalledTimes(3);
  });

  it('la caché ya está limpia cuando el componente hace su refetch', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));
    await api.get('/users');

    // Patrón real: refetch dentro del .then del POST.
    await api.post('/users', {}).then(() => api.get('/users'));

    expect(fetch).toHaveBeenCalledTimes(3);
  });
});

describe('clearCache', () => {
  it('vacía todo', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [] }));
    await api.get('/users');
    await api.get('/plans');

    clearCache();

    await api.get('/users');
    await api.get('/plans');
    expect(fetch).toHaveBeenCalledTimes(4);
  });
});

describe('getJSON', () => {
  it('devuelve los datos parseados', async () => {
    fetch.mockResolvedValue(jsonResponse({ data: [{ id: 1 }] }));
    await expect(getJSON('/users')).resolves.toEqual({ data: [{ id: 1 }] });
  });

  it('lanza con el mensaje del backend', async () => {
    fetch.mockResolvedValue(jsonResponse({ error: 'Stock insuficiente' }, 400));
    await expect(getJSON('/products')).rejects.toThrow('Stock insuficiente');
  });
});
