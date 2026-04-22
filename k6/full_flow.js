import encoding from "k6/encoding";
import http from "k6/http";
import ws from "k6/ws";
import { check } from "k6";
import exec from "k6/execution";
import { createHMAC } from "k6/crypto";

const baseUrl = __ENV.APP_URL || "http://localhost:8080";
const wsBaseUrl = baseUrl.replace(/^http/, "ws");
const webhookSecret = __ENV.WEBHOOK_SECRET || "dev-webhook-secret";
const jwtSecret = __ENV.JWT_SECRET || "dev-jwt-secret";
const wsTimeoutMs = Number(__ENV.WS_TIMEOUT_MS || 3000);
const expected200 = http.expectedStatuses(200);
const expected200or201 = http.expectedStatuses(200, 201);
const expected200or404 = http.expectedStatuses(200, 404);

export const options = {
  scenarios: {
    notification_lifecycle: {
      executor: "shared-iterations",
      vus: Number(__ENV.FLOW_VUS || 1),
      iterations: Number(__ENV.FLOW_ITERATIONS || 10),
      exec: "notificationLifecycle",
    },
    websocket_delivery: {
      executor: "shared-iterations",
      vus: Number(__ENV.WS_FLOW_VUS || 1),
      iterations: Number(__ENV.WS_FLOW_ITERATIONS || 5),
      exec: "websocketDelivery",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<1000"],
    checks: ["rate>0.95"],
  },
};

function base64UrlEncode(value) {
  return encoding.b64encode(value).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
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

function signWebhook(body) {
  const mac = createHMAC("sha256", webhookSecret);
  mac.update(body);
  return `sha256=${mac.digest("hex")}`;
}

function calculateCPFCheckDigit(digits) {
  let factor = digits.length + 1;
  let sum = 0;
  for (const digit of digits) {
    sum += digit * factor;
    factor -= 1;
  }

  const remainder = sum % 11;
  return remainder < 2 ? 0 : 11 - remainder;
}

function generateCPF(seed) {
  let base = String((123456789 + seed) % 1000000000).padStart(9, "0");
  if (/^(\d)\1{8}$/.test(base)) {
    base = "123456789";
  }

  const digits = base.split("").map(Number);
  const digitOne = calculateCPFCheckDigit(digits);
  const digitTwo = calculateCPFCheckDigit([...digits, digitOne]);
  return `${base}${digitOne}${digitTwo}`;
}

function iterationSeed(offset) {
  return exec.vu.idInTest * 100000 + exec.scenario.iterationInTest + offset;
}

function buildFlowContext(kind) {
  const offset = kind === "ws" ? 500000 : 0;
  const seed = iterationSeed(offset);
  const cpf = generateCPF(seed);
  const kindCode = kind === "ws" ? "ws" : "lf";
  const chamadoId = `CH-K6-${kindCode}-${seed}-${Date.now()}`;
  const token = createDevJWT(cpf);
  const payload = {
    chamado_id: chamadoId,
    tipo: "status_change",
    cpf,
    status_anterior: "em_analise",
    status_novo: "em_execucao",
    titulo: "Buraco na Rua - Fluxo completo",
    descricao: `Fluxo ${kind} para CPF ${cpf}`,
    timestamp: new Date().toISOString(),
  };

  return {
    chamadoId,
    cpf,
    token,
    body: JSON.stringify(payload),
  };
}

function parseJSON(response) {
  try {
    return response.json();
  } catch (_) {
    return null;
  }
}

function authHeaders(token) {
  return {
    Authorization: `Bearer ${token}`,
  };
}

function webhookHeaders(body) {
  return {
    "Content-Type": "application/json",
    "X-Signature-256": signWebhook(body),
  };
}

function findNotificationByChamadoId(page, chamadoId) {
  if (!page || !Array.isArray(page.data)) {
    return null;
  }

  for (const notification of page.data) {
    if (notification.chamado_id === chamadoId) {
      return notification;
    }
  }

  return null;
}

function logUnexpectedResponse(step, response, extra = {}) {
  const details = {
    step,
    status: response && response.status,
    body: response && response.body,
    ...extra,
  };

  console.error(`[full_flow] ${JSON.stringify(details)}`);
}

export function notificationLifecycle() {
  const flow = buildFlowContext("lifecycle");
  const headers = authHeaders(flow.token);

  const initialUnread = http.get(`${baseUrl}/api/v1/notifications/unread-count`, {
    headers,
    tags: { endpoint: "initial_unread_count" },
  });
  const initialUnreadBody = parseJSON(initialUnread);
  check(initialUnread, {
    "initial unread count status ok": (r) => r.status === 200,
    "initial unread count is zero": () => initialUnreadBody && initialUnreadBody.count === 0,
  });

  const webhookRes = http.post(`${baseUrl}/api/v1/webhooks/status-change`, flow.body, {
    headers: webhookHeaders(flow.body),
    tags: { endpoint: "webhook_lifecycle" },
    responseCallback: expected200or201,
  });
  const webhookBody = parseJSON(webhookRes);
  if (webhookRes.status !== 201) {
    logUnexpectedResponse("lifecycle_webhook", webhookRes, {
      chamado_id: flow.chamadoId,
      cpf: flow.cpf,
    });
  }
  check(webhookRes, {
    "lifecycle webhook accepted": (r) => r.status === 201,
    "lifecycle webhook returns notification id": () => webhookBody && webhookBody.notification_id,
  });

  const unreadAfterWebhook = http.get(`${baseUrl}/api/v1/notifications/unread-count`, {
    headers,
    tags: { endpoint: "unread_count_after_webhook" },
  });
  const unreadAfterWebhookBody = parseJSON(unreadAfterWebhook);
  check(unreadAfterWebhook, {
    "unread count after webhook status ok": (r) => r.status === 200,
    "unread count after webhook is one": () => unreadAfterWebhookBody && unreadAfterWebhookBody.count === 1,
  });

  const listRes = http.get(`${baseUrl}/api/v1/notifications?limit=10`, {
    headers,
    tags: { endpoint: "notifications_list_lifecycle" },
    responseCallback: expected200,
  });
  const listBody = parseJSON(listRes);
  const notification = findNotificationByChamadoId(listBody, flow.chamadoId);
  if (!notification) {
    logUnexpectedResponse("lifecycle_list_missing_notification", listRes, {
      chamado_id: flow.chamadoId,
      response_json: listBody,
    });
  }
  check(listRes, {
    "lifecycle notifications list ok": (r) => r.status === 200,
    "lifecycle notification found in list": () => !!notification,
    "lifecycle notification initially unread": () => notification && !notification.read_at,
  });

  let markReadOk = false;
  let secondMarkReadRejected = false;

  if (notification) {
    const markReadRes = http.patch(`${baseUrl}/api/v1/notifications/${notification.id}/read`, "", {
      headers,
      tags: { endpoint: "mark_read" },
      responseCallback: expected200or404,
    });
    if (markReadRes.status !== 200) {
      logUnexpectedResponse("mark_read", markReadRes, {
        chamado_id: flow.chamadoId,
        notification_id: notification.id,
      });
    } else {
      markReadOk = true;
    }

    const secondMarkReadRes = http.patch(`${baseUrl}/api/v1/notifications/${notification.id}/read`, "", {
      headers,
      tags: { endpoint: "mark_read_again" },
      responseCallback: expected200or404,
    });
    secondMarkReadRejected = secondMarkReadRes.status === 404;
  }

  check(null, {
    "mark read status ok": () => markReadOk,
    "second mark read rejected": () => secondMarkReadRejected,
  });

  const unreadAfterRead = http.get(`${baseUrl}/api/v1/notifications/unread-count`, {
    headers,
    tags: { endpoint: "unread_count_after_read" },
  });
  const unreadAfterReadBody = parseJSON(unreadAfterRead);
  check(unreadAfterRead, {
    "unread count after read status ok": (r) => r.status === 200,
    "unread count after read is zero": () => unreadAfterReadBody && unreadAfterReadBody.count === 0,
  });

  const duplicateWebhookRes = http.post(`${baseUrl}/api/v1/webhooks/status-change`, flow.body, {
    headers: webhookHeaders(flow.body),
    tags: { endpoint: "duplicate_webhook" },
    responseCallback: expected200or201,
  });
  const duplicateWebhookBody = parseJSON(duplicateWebhookRes);
  if (duplicateWebhookRes.status !== 200) {
    logUnexpectedResponse("duplicate_webhook", duplicateWebhookRes, {
      chamado_id: flow.chamadoId,
      cpf: flow.cpf,
    });
  }
  check(duplicateWebhookRes, {
    "duplicate webhook acknowledged": (r) => r.status === 200,
    "duplicate webhook message returned": () =>
      duplicateWebhookBody && duplicateWebhookBody.message === "webhook already processed for chamado",
  });
}

export function websocketDelivery() {
  const flow = buildFlowContext("ws");
  const headers = authHeaders(flow.token);

  let webhookAccepted = false;
  let deliveredNotification = null;

  const response = ws.connect(`${wsBaseUrl}/ws`, { headers }, function (socket) {
    socket.on("open", function () {
      const webhookRes = http.post(`${baseUrl}/api/v1/webhooks/status-change`, flow.body, {
        headers: webhookHeaders(flow.body),
        tags: { endpoint: "webhook_ws_delivery" },
        responseCallback: expected200or201,
      });
      webhookAccepted = webhookRes.status === 201;
      if (!webhookAccepted) {
        logUnexpectedResponse("websocket_webhook", webhookRes, {
          chamado_id: flow.chamadoId,
          cpf: flow.cpf,
        });
      }
    });

    socket.on("message", function (message) {
      try {
        deliveredNotification = JSON.parse(message);
      } catch (_) {
        deliveredNotification = null;
      }
      socket.close();
    });

    socket.setTimeout(function () {
      socket.close();
    }, wsTimeoutMs);
  });

  check(response, {
    "websocket handshake ok": (r) => r && r.status === 101,
  });
  check(null, {
    "websocket webhook accepted": () => webhookAccepted,
    "websocket notification delivered": () =>
      deliveredNotification &&
      deliveredNotification.chamado_id === flow.chamadoId &&
      deliveredNotification.status_novo === "em_execucao" &&
      deliveredNotification.titulo === "Buraco na Rua - Fluxo completo",
  });
}
