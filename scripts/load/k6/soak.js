
import { get, endpoints } from './_common.js';
import { sleep } from 'k6';

// Soak test: stability over time. Keep thresholds similar to combined.
export const options = {
  thresholds: {
    http_req_failed: ['rate<=0.005'],
    http_req_duration: ['p(95)<=450', 'p(99)<=1800'],
  },
};

export default function () {
  const ep = endpoints();

  // Realistic "HA-like" pattern: read core metrics frequently.
  get(ep.load, 'load');
  get(ep.genmix, 'generation-mix');
  get(ep.balance, 'balance');

  // Longer sleep approximates periodic polling; you can override by editing.
  sleep(0.8);

}
