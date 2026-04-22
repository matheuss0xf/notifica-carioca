import encoding from "k6/encoding";
import http from "k6/http";
import { check, sleep } from "k6";
import exec from "k6/execution";
import { createHMAC } from "k6/crypto";

const baseUrl = __ENV.APP_URL || "http://localhost:8080";
const webhookSecret = __ENV.WEBHOOK_SECRET || "dev-webhook-secret";
const jwtSecret = __ENV.JWT_SECRET || "dev-jwt-secret";

export const options = {
  scenarios: {
    webhook_burst: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.WEBHOOK_RATE || 20),
      timeUnit: "1s",
      duration: __ENV.WEBHOOK_DURATION || "1m",
      preAllocatedVUs: Number(__ENV.WEBHOOK_PREALLOCATED_VUS || 20),
      maxVUs: Number(__ENV.WEBHOOK_MAX_VUS || 80),
      exec: "webhookBurst",
    },
    notifications_read: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: __ENV.READ_RAMP_UP || "15s", target: Number(__ENV.READ_TARGET_VUS || 10) },
        { duration: __ENV.READ_STEADY || "30s", target: Number(__ENV.READ_TARGET_VUS || 10) },
        { duration: __ENV.READ_RAMP_DOWN || "15s", target: 0 },
      ],
      exec: "notificationsRead",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<1000"],
    checks: ["rate>0.95"],
  },
};

function buildWebhookPayload() {
  const uid = `${exec.scenario.name}-${exec.vu.idInTest}-${exec.scenario.iterationInTest}-${Date.now()}`;
  return {
    chamado_id: `CH-K6-${uid}`,
    tipo: "status_change",
    cpf: "529.982.247-25",
    status_anterior: "em_analise",
    status_novo: "em_execucao",
    titulo: "Buraco na Rua - Atualizacao",
    descricao: "Equipe designada para reparo na Rua das Laranjeiras, 100",
    timestamp: new Date().toISOString(),
  };
}

function signWebhook(body) {
  const mac = createHMAC("sha256", webhookSecret);
  mac.update(body);
  return `sha256=${mac.digest("hex")}`;
}

function base64UrlEncode(value) {
  return encoding
    .b64encode(value)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

function toBase64Url(value) {
  return value.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function createDevJWT(cpf) {
  const header = base64UrlEncode(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = base64UrlEncode(
    JSON.stringify({
      preferred_username: cpf,
      iat: Math.floor(Date.now() / 1000),
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  );
  const unsignedToken = `${header}.${payload}`;
  const mac = createHMAC("sha256", jwtSecret);
  mac.update(unsignedToken);
  const signature = toBase64Url(mac.digest("base64"));
  return `${unsignedToken}.${signature}`;
}

export function webhookBurst() {
  const body = JSON.stringify(buildWebhookPayload());
  const res = http.post(`${baseUrl}/api/v1/webhooks/status-change`, body, {
    headers: {
      "Content-Type": "application/json",
      "X-Signature-256": signWebhook(body),
    },
    tags: {
      endpoint: "webhook",
    },
  });

  check(res, {
    "webhook accepted": (r) => r.status === 201 || r.status === 200,
  });
}

export function notificationsRead() {
  const token = createDevJWT("52998224725");
  const headers = {
    Authorization: `Bearer ${token}`,
  };

  const listRes = http.get(`${baseUrl}/api/v1/notifications?limit=10`, {
    headers,
    tags: {
      endpoint: "notifications_list",
    },
  });

  check(listRes, {
    "notifications list ok": (r) => r.status === 200,
  });

  const countRes = http.get(`${baseUrl}/api/v1/notifications/unread-count`, {
    headers,
    tags: {
      endpoint: "notifications_unread_count",
    },
  });

  check(countRes, {
    "notifications unread count ok": (r) => r.status === 200,
  });

  sleep(1);
}
