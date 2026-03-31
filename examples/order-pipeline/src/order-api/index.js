const express = require('express');
const mysql = require('mysql2/promise');
const amqp = require('amqplib');
const { randomUUID } = require('crypto');

const app = express();
app.use(express.json());

// ---------------------------------------------------------------------------
// Database connection
// ---------------------------------------------------------------------------

const db = mysql.createPool({
  host: process.env.DB_HOST || 'localhost',
  port: Number(process.env.DB_PORT) || 3306,
  user: process.env.DB_USER || 'cx',
  password: process.env.DB_PASSWORD || 'cx',
  database: process.env.DB_NAME || 'cx',
  waitForConnections: true,
});

// ---------------------------------------------------------------------------
// RabbitMQ connection
// ---------------------------------------------------------------------------

let channel;

async function connectRabbitMQ(retries = 10, delayMs = 3000) {
  for (let i = 1; i <= retries; i++) {
    try {
      const conn = await amqp.connect(
        process.env.RABBITMQ_URL || 'amqp://localhost',
      );
      channel = await conn.createChannel();
      break;
    } catch (err) {
      if (i === retries) throw err;
      console.log(
        `RabbitMQ not ready (attempt ${i}/${retries}), retrying in ${delayMs}ms...`,
      );
      await new Promise((r) => setTimeout(r, delayMs));
    }
  }

  // Assert the exchange and all queues so the topology exists regardless of
  // which service starts first. This is idempotent — other services calling
  // assertQueue for the same queue is safe.
  await channel.assertExchange('order-events', 'topic', { durable: true });

  const queues = ['inventory-orders', 'notification-orders', 'assert-orders'];

  for (const q of queues) {
    await channel.assertQueue(q, { durable: true });
    await channel.bindQueue(q, 'order-events', 'order.#');
  }

  console.log('RabbitMQ connected — topology ready');
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

// POST /orders — place a new order
app.post('/orders', async (req, res) => {
  const { productId } = req.body;

  if (!productId) {
    return res.status(400).json({ error: 'productId is required' });
  }

  // Check stock
  const [rows] = await db.execute(
    'SELECT id, name, stock FROM products WHERE id = ?',
    [productId],
  );

  if (rows.length === 0) {
    return res.status(404).json({ error: 'product not found' });
  }

  const product = rows[0];

  if (product.stock <= 0) {
    return res.status(409).json({ error: 'out of stock' });
  }

  // Persist order
  const orderId = randomUUID();

  await db.execute(
    'INSERT INTO orders (id, product_id, status) VALUES (?, ?, ?)',
    [orderId, productId, 'placed'],
  );

  // Decrement stock atomically
  await db.execute('UPDATE products SET stock = stock - 1 WHERE id = ?', [
    productId,
  ]);

  // Publish event — downstream services react asynchronously
  const payload = { event: 'order.placed', orderId, productId };
  channel.publish(
    'order-events',
    'order.placed',
    Buffer.from(JSON.stringify(payload)),
    { persistent: true },
  );

  return res.status(201).json({ orderId, productId, status: 'placed' });
});

// GET /orders/:id
app.get('/orders/:id', async (req, res) => {
  const [rows] = await db.execute(
    'SELECT id, product_id AS productId, status, created_at AS createdAt FROM orders WHERE id = ?',
    [req.params.id],
  );

  if (rows.length === 0) {
    return res.status(404).json({ error: 'order not found' });
  }

  return res.json(rows[0]);
});

// GET /products/:id — useful for verifying stock levels in tests
app.get('/products/:id', async (req, res) => {
  const [rows] = await db.execute(
    'SELECT id, name, stock FROM products WHERE id = ?',
    [req.params.id],
  );

  if (rows.length === 0) {
    return res.status(404).json({ error: 'product not found' });
  }

  return res.json(rows[0]);
});

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

async function start() {
  await connectRabbitMQ();

  const port = process.env.PORT || 3000;
  app.listen(port, () => console.log(`order-api listening on :${port}`));
}

start().catch((err) => {
  console.error('Startup failed:', err);
  process.exit(1);
});
