
import { get, think, endpoints } from './_common.js';

export const options = {
  thresholds: {
    http_req_failed: ['rate<=0.005'],      // <=0.5% errors
    http_req_duration: ['p(95)<=250', 'p(99)<=750'],
  },
};

export default function () {
  const ep = endpoints();
  get(ep.load, 'load');
  think();
}
