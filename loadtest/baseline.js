import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m',  target: 200 },
    { duration: '30s', target: 500 },
    { duration: '1m',  target: 500 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const body = JSON.stringify({
    channel: 'email',
    to: 'load@example.com',
    subject: 'Load test',
    body: 'k6 baseline',
  });
  const res = http.post('http://localhost:8080/v1/notifications', body, {
    headers: { 'Content-Type': 'application/json', 'Idempotency-Key': uuidv4() },
  });
  check(res, { 'status is 202': r => r.status === 202 });
  sleep(0.1);
}
