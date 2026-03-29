const amqp = require('amqplib');
const http = require('http');
const https = require('https');

const WEBHOOK_URL = process.env.WEBHOOK_URL;

// ---------------------------------------------------------------------------
// Webhook helper
// ---------------------------------------------------------------------------

function fireWebhook(url, payload) {
  const body = JSON.stringify(payload);
  const lib = url.startsWith('https') ? https : http;
  const urlObj = new URL(url);

  const req = lib.request(
    {
      hostname: urlObj.hostname,
      port: urlObj.port,
      path: urlObj.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(body),
      },
    },
    (res) => res.resume(),
  );

  req.on('error', (err) => console.error('Webhook error:', err.message));
  req.write(body);
  req.end();
}

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

async function start() {
  const conn = await amqp.connect(
    process.env.RABBITMQ_URL || 'amqp://localhost',
  );
  const ch = await conn.createChannel();

  await ch.assertExchange('order-events', 'topic', { durable: true });
  await ch.assertQueue('notification-orders', { durable: true });
  await ch.bindQueue('notification-orders', 'order-events', 'order.#');

  ch.consume('notification-orders', async (msg) => {
    if (!msg) return;

    try {
      const { event, orderId, productId } = JSON.parse(msg.content.toString());

      if (event === 'order.placed' && WEBHOOK_URL) {
        fireWebhook(`${WEBHOOK_URL}notify`, { event, orderId, productId });
        console.log(
          `notification-service: fired notify webhook for order ${orderId}`,
        );
      }

      ch.ack(msg);
    } catch (err) {
      console.error('notification-service: message processing error:', err);
      ch.nack(msg, false, false);
    }
  });

  console.log('notification-service: consuming from notification-orders');
}

start().catch((err) => {
  console.error('Startup failed:', err);
  process.exit(1);
});
