
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8000';
const COUNTRY = __ENV.COUNTRY || 'NL';

export function get(url, name) {
  const res = http.get(url, { tags: { endpoint: name } });

  // Basic health checks (kept strict; adjust if your API returns 204 etc.)
  check(res, {
    [`${name}: status is 200`]: (r) => r.status === 200,
  });

  return res;
}

export function think() {
  // Small think-time prevents unrealistic "machine gun" patterns.
  sleep(0.2);
}

export function endpoints() {
  return {
    load: `${BASE}/api/v1/load?country=${COUNTRY}`,
    genmix: `${BASE}/api/v1/generation-mix?country=${COUNTRY}`,
    balance: `${BASE}/api/v1/balance?country=${COUNTRY}`,
  };
}
