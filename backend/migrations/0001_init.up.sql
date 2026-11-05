-- T1.1 建表迁移。数据模型见 WF§10，补充约束见 WF§17.4。
-- 业务规则在 Go 中实现，数据库只做最后一道兜底（不使用触发器和存储过程）。

CREATE TYPE user_role       AS ENUM ('agent', 'supervisor');
CREATE TYPE ticket_status   AS ENUM ('open', 'in_progress', 'resolved', 'closed');
CREATE TYPE ticket_priority AS ENUM ('urgent', 'high', 'normal', 'low');

CREATE TABLE users (
  id         BIGSERIAL PRIMARY KEY,
  name       TEXT NOT NULL,
  role       user_role NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tickets (
  id               BIGSERIAL PRIMARY KEY,
  title            TEXT NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  description      TEXT NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 5000),
  customer_name    TEXT NOT NULL CHECK (length(btrim(customer_name)) BETWEEN 1 AND 100),
  customer_contact TEXT NOT NULL CHECK (length(btrim(customer_contact)) BETWEEN 1 AND 200),
  priority         ticket_priority NOT NULL,
  status           ticket_status NOT NULL DEFAULT 'open',
  assignee_id      BIGINT REFERENCES users(id),
  created_by       BIGINT NOT NULL REFERENCES users(id),
  sla_due_at       TIMESTAMPTZ NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- open 必无处理人，非 open 必有处理人（WF§3）
  CONSTRAINT open_iff_unassigned CHECK ((status = 'open') = (assignee_id IS NULL)),
  -- WF§17.4
  CONSTRAINT sla_after_created CHECK (sla_due_at > created_at),
  CONSTRAINT updated_not_before_created CHECK (updated_at >= created_at)
);

CREATE TABLE ticket_comments (
  id         BIGSERIAL PRIMARY KEY,
  ticket_id  BIGINT NOT NULL REFERENCES tickets(id),
  author_id  BIGINT NOT NULL REFERENCES users(id),
  body       TEXT NOT NULL CHECK (length(btrim(body)) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_events (
  id         BIGSERIAL PRIMARY KEY,
  ticket_id  BIGINT NOT NULL REFERENCES tickets(id),
  actor_id   BIGINT NOT NULL REFERENCES users(id),
  type       TEXT NOT NULL CHECK (type IN
             ('created','claimed','assigned','reassigned','status_changed','commented')),
  payload    JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_status_created ON tickets (status, created_at DESC);
CREATE INDEX idx_tickets_assignee       ON tickets (assignee_id);
CREATE INDEX idx_tickets_priority       ON tickets (priority);
CREATE INDEX idx_comments_ticket        ON ticket_comments (ticket_id, created_at);
CREATE INDEX idx_events_ticket          ON ticket_events (ticket_id, created_at);
