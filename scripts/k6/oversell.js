// Oversell load test. Run it against a server whose monitor (product 3) has exactly one in stock.
//
//   docker compose up -d --wait
//   RATE_LIMIT_RPS=100000 RATE_LIMIT_BURST=100000 DATABASE_URL=... REDIS_URL=... go run ./cmd/api   # a load test comes from one IP
//   k6 run scripts/k6/oversell.js
//
// Every virtual user is a separate browser (own cookie jar, so own session and own cart): it puts one monitor in its
// cart and checks out, all at the same moment. With the transaction in place exactly one checkout may return 201.
// To run it again, put the stock back first (see the README).
import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

const BASE = __ENV.BASE_URL || "http://localhost:8080";
const MONITOR = 3;
const bought = new Counter("orders_bought");
const soldOut = new Counter("orders_sold_out");

export const options = {
  scenarios: {
    stampede: { executor: "per-vu-iterations", vus: 50, iterations: 1, maxDuration: "30s" },
  },
  thresholds: {
    // Exactly one winner. (k6 evaluates thresholds on the counter's final value.)
    orders_bought: ["count==1"],
    orders_sold_out: ["count==49"],
  },
};

export default function () {
  const put = http.put(`${BASE}/cart/items/${MONITOR}`, JSON.stringify({ quantity: 1 }), {
    headers: { "Content-Type": "application/json" },
  });
  check(put, { "added to cart": (r) => r.status === 200 });

  const res = http.post(`${BASE}/checkout`);
  if (res.status === 201) bought.add(1);
  else if (res.status === 409) soldOut.add(1);
  check(res, { "201 or 409, never a 500": (r) => r.status === 201 || r.status === 409 });
}
