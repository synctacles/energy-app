
import { get, think, endpoints } from './_common.js';

export const options = {
  thresholds: {
    http_req_failed: ['rate<=0.005'],
    http_req_duration: ['p(95)<=400', 'p(99)<=1500'],
  },
};

export default function () {
  const ep = endpoints();
  // parallel-ish: these fire back-to-back inside the same VU loop
  get(ep.load, 'load');
  get(ep.genmix, 'generation-mix');
  get(ep.balance, 'balance');
  think();
}
