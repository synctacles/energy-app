
import { get, think, endpoints } from './_common.js';

export const options = {
  thresholds: {
    http_req_failed: ['rate<=0.005'],
    http_req_duration: ['p(95)<=350', 'p(99)<=1200'], // heavier query; slightly looser defaults
  },
};

export default function () {
  const ep = endpoints();
  get(ep.genmix, 'generation-mix');
  think();
}
