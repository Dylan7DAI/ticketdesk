-- 删除全部 seed 数据并重置序列。按外键依赖的反序删除。
-- 序列重置为未使用状态：下一个 nextval 返回 1。
DELETE FROM ticket_events;
DELETE FROM ticket_comments;
DELETE FROM tickets;
DELETE FROM users;

SELECT setval(pg_get_serial_sequence('ticket_events', 'id'),   1, false);
SELECT setval(pg_get_serial_sequence('ticket_comments', 'id'), 1, false);
SELECT setval(pg_get_serial_sequence('tickets', 'id'),         1, false);
SELECT setval(pg_get_serial_sequence('users', 'id'),           1, false);
