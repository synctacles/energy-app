
import { get, think, endpoints } from './_common.js';

export const options = {
  thresholds: {
    http_req_failed: ['rate<=0.005'],
    http_req_duration: ['p(95)<=300', 'p(99)<=900'],
  },
};

export default function () {
  const ep = endpoints();
  get(ep.balance, 'balance');
  think();
}
