const mysql = require('mysql2/promise');
const { createClient } = require('redis');
const amqp = require('amqplib');
const http = require('http');
const https = require('https');

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const LOW_STOCK_THRESHOLD = Number(process.env.LOW_STOCK_THRESHOLD ?? 2);
const WEBHOOK_URL = process.env.WEBHOOK_URL;

// ---------------------------------------------------------------------------
// Database + cache connections
// ---------------------------------------------------------------------------

const db = mysql.createPool({
  host: process.env.DB_HOST || 'localhost',
  port: Number(process.env.DB_PORT) || 3306,
  user: process.env.DB_USER || 'cx',
  password: process.env.DB_PASSWORD || 'cx',
  database: process.env.DB_NAME || 'cx',
  waitForConnections: true,
});

const cache = createClient({
  url: process.env.REDIS_URL || 'redis://localhost:6379',
});

cache.on('error', (err) => console.error('Redis error:', err));

// ---------------------------------------------------------------------------
// Event handlers
// ---------------------------------------------------------------------------

async function handleOrderPlaced({ orderId, productId }) {
  // Read current stock from the source of truth
  const [rows] = await db.execute('SELECT stock FROM products WHERE id = ?', [
    productId,
  ]);

  if (rows.length === 0) {
    console.warn(`inventory-service: product ${productId} not found`);
    return;
  }

  const { stock } = rows[0];

  // Cache the current stock level so read paths skip the DB
  await cache.set(`stock:${productId}`, JSON.stringify({ stock }));
  console.log(`inventory-service: cached stock:${productId} = ${stock}`);

  // Alert when stock drops below threshold
  if (stock <= LOW_STOCK_THRESHOLD && WEBHOOK_URL) {
    fireWebhook(`${WEBHOOK_URL}low-stock`, {
      event: 'low-stock',
      productId,
      stock,
      orderId,
    });
    console.log(`inventory-service: fired low-stock webhook (stock=${stock})`);
  }
}

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
  await cache.connect();
  console.log('inventory-service: Redis connected');

  const conn = await amqp.connect(
    process.env.RABBITMQ_URL || 'amqp://localhost',
  );
  const ch = await conn.createChannel();

  await ch.assertExchange('order-events', 'topic', { durable: true });
  await ch.assertQueue('inventory-orders', { durable: true });
  await ch.bindQueue('inventory-orders', 'order-events', 'order.#');

  ch.consume('inventory-orders', async (msg) => {
    if (!msg) return;

    try {
      const payload = JSON.parse(msg.content.toString());

      if (payload.event === 'order.placed') {
        await handleOrderPlaced(payload);
      }

      ch.ack(msg);
    } catch (err) {
      console.error('inventory-service: message processing error:', err);
      ch.nack(msg, false, false);
    }
  });

  console.log('inventory-service: consuming from inventory-orders');
}

start().catch((err) => {
  console.error('Startup failed:', err);
  process.exit(1);
});
