const express = require('express');
const mysql = require('mysql2/promise');
const { createClient } = require('redis');
const { randomUUID } = require('crypto');
const http = require('http');
const https = require('https');

const app = express();
app.use(express.json());

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
// Routes
// ---------------------------------------------------------------------------

// POST /users  — create a new user
app.post('/users', async (req, res) => {
  const { name, email } = req.body;

  if (!name || !email) {
    return res.status(400).json({ error: 'name and email are required' });
  }

  const id = randomUUID();

  await db.execute(
    'INSERT INTO users (id, name, email) VALUES (?, ?, ?)',
    [id, name, email],
  );

  await cache.set(`user:${id}`, JSON.stringify({ id, name, email }));

  // Fire webhook if WEBHOOK_URL is configured (best-effort, non-blocking).
  const webhookUrl = process.env.WEBHOOK_URL;

  if (webhookUrl) {
    fireWebhook(webhookUrl, { event: 'user.created', userId: id, name, email });
  }

  return res.status(201).json({ id, name, email });
});

// GET /users/:id  — fetch a user by id
app.get('/users/:id', async (req, res) => {
  const { id } = req.params;

  const [rows] = await db.execute('SELECT id, name, email FROM users WHERE id = ?', [id]);

  if (rows.length === 0) {
    return res.status(404).json({ error: 'user not found' });
  }

  return res.json(rows[0]);
});

// ---------------------------------------------------------------------------
// Helpers
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
  console.log('Redis connected');

  const port = process.env.PORT || 3000;
  app.listen(port, () => console.log(`users-api listening on :${port}`));
}

start().catch((err) => {
  console.error('Startup failed:', err);
  process.exit(1);
});
