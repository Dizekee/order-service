CREATE TABLE products (
    sku TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    price INT NOT NULL,
    currency TEXT DEFAULT 'RUB'
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku TEXT NOT NULL REFERENCES products(sku),
    amount INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE webhook_events (
    event_id TEXT PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id),
    status TEXT NOT NULL,
    amount INT,
    currency TEXT,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE issued_codes (
    id SERIAL PRIMARY KEY,
    order_id UUID UNIQUE NOT NULL REFERENCES orders(id),
    code TEXT NOT NULL,
    supplier_id TEXT,
    request_id TEXT,
    issued_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_webhook_events_order_id ON webhook_events(order_id);