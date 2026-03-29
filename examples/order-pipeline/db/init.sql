-- Products catalogue with initial stock
CREATE TABLE IF NOT EXISTS products (
  id         VARCHAR(36)  NOT NULL PRIMARY KEY,
  name       VARCHAR(255) NOT NULL,
  stock      INT          NOT NULL DEFAULT 0,
  INDEX idx_stock (stock)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Placed orders
CREATE TABLE IF NOT EXISTS orders (
  id         VARCHAR(36)  NOT NULL PRIMARY KEY,
  product_id VARCHAR(36)  NOT NULL,
  status     VARCHAR(50)  NOT NULL DEFAULT 'placed',
  created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_product (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed data — two products, one well-stocked and one nearly empty
INSERT INTO products (id, name, stock) VALUES
  ('prod-001', 'Widget A', 10),
  ('prod-002', 'Widget B', 1);
