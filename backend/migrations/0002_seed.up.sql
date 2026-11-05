-- T1.2 演示数据。用户见 WF§1，工单见 WF§10 演示工单表。
-- 所有时间相对 now() 构造，sla_due_at = created_at + SLA 时长（WF§8）。
-- 主键显式指定，末尾用 setval 把序列推到最大 ID 之后。

INSERT INTO users (id, name, role) VALUES
  (1, '主管 王敏', 'supervisor'),
  (2, '客服 李雷', 'agent'),
  (3, '客服 韩梅', 'agent'),
  (4, '客服 张伟', 'agent');

-- 6 张工单覆盖 4 种状态；updated_at 等于该工单最后一条事件的时间。
INSERT INTO tickets
  (id, title, description, customer_name, customer_contact, priority, status,
   assignee_id, created_by, sla_due_at, created_at, updated_at)
VALUES
  -- 1 紧急、待领取、已逾期（3h 前创建，SLA 2h）
  (1, '支付失败无法下单', '客户在结算页点击支付后提示“支付失败”，多次重试均无法下单。',
   '陈晨', 'chenchen@example.com', 'urgent', 'open', NULL, 2,
   now() - interval '3 hours' + interval '2 hours',
   now() - interval '3 hours', now() - interval '3 hours'),
  -- 2 普通、待领取、未逾期，用于演示并发领取
  (2, '发票抬头修改', '订单已开具发票，客户希望把抬头改为公司全称。',
   '周静', '13800001111', 'normal', 'open', NULL, 2,
   now() - interval '1 hour' + interval '24 hours',
   now() - interval '1 hour', now() - interval '1 hour'),
  -- 3 高、处理中（李雷）、已逾期（10h 前创建，SLA 8h）
  (3, '物流长时间未更新', '订单显示已发货 5 天，物流信息一直停留在揽收。',
   '吴磊', 'wulei@example.com', 'high', 'in_progress', 2, 1,
   now() - interval '10 hours' + interval '8 hours',
   now() - interval '10 hours', now() - interval '9 hours'),
  -- 4 高、处理中（韩梅）、未逾期，用于演示改派
  (4, '账号无法登录', '客户反馈输入正确密码后提示“账号异常”，无法登录。',
   '郑洁', '13900002222', 'high', 'in_progress', 3, 1,
   now() - interval '2 hours' + interval '8 hours',
   now() - interval '2 hours', now() - interval '90 minutes'),
  -- 5 普通、已解决（李雷）。SLA 已过但已解决，不显示逾期
  (5, '退款进度查询', '客户申请退款 3 天，想确认退款是否已到账。',
   '孙悦', 'sunyue@example.com', 'normal', 'resolved', 2, 3,
   now() - interval '30 hours' + interval '24 hours',
   now() - interval '30 hours', now() - interval '26 hours'),
  -- 6 低、已关闭（张伟），终态
  (6, '修改收货地址', '客户下单后发现地址填错，希望改为新地址。',
   '马丽', '13700003333', 'low', 'closed', 4, 1,
   now() - interval '5 days' + interval '72 hours',
   now() - interval '5 days', now() - interval '4 days');

-- 评论（至少 2 张工单有评论）：工单 3、4、5 各一条。评论不改 tickets.updated_at。
INSERT INTO ticket_comments (id, ticket_id, author_id, body, created_at) VALUES
  (1, 3, 2, '已联系快递公司核实，对方称包裹滞留在中转站，正在催促发出。', now() - interval '8 hours'),
  (2, 4, 3, '已让客户尝试重置密码，等待客户反馈结果。', now() - interval '80 minutes'),
  (3, 5, 2, '已在支付渠道确认退款已发起，预计 1～3 个工作日到账。', now() - interval '27 hours');

-- 历史事件：与各工单当前状态一致，payload 结构见 WF§6。
-- 同一张工单的事件按 created_at 升序即为真实发生顺序。
INSERT INTO ticket_events (ticket_id, actor_id, type, payload, created_at) VALUES
  -- 工单 1：open
  (1, 2, 'created', '{"priority":"urgent"}', now() - interval '3 hours'),
  -- 工单 2：open
  (2, 2, 'created', '{"priority":"normal"}', now() - interval '1 hour'),
  -- 工单 3：open → in_progress（李雷领取）
  (3, 1, 'created', '{"priority":"high"}', now() - interval '10 hours'),
  (3, 2, 'claimed', '{"assignee_id":2}', now() - interval '9 hours'),
  (3, 2, 'commented', '{"comment_id":1,"excerpt":"已联系快递公司核实，对方称包裹滞留在中转站，正在催促发出。"}',
   now() - interval '8 hours'),
  -- 工单 4：open → in_progress（王敏分配给韩梅）
  (4, 1, 'created', '{"priority":"high"}', now() - interval '2 hours'),
  (4, 1, 'assigned', '{"assignee_id":3}', now() - interval '90 minutes'),
  (4, 3, 'commented', '{"comment_id":2,"excerpt":"已让客户尝试重置密码，等待客户反馈结果。"}',
   now() - interval '80 minutes'),
  -- 工单 5：open → in_progress → resolved
  (5, 3, 'created', '{"priority":"normal"}', now() - interval '30 hours'),
  (5, 2, 'claimed', '{"assignee_id":2}', now() - interval '29 hours'),
  (5, 2, 'commented', '{"comment_id":3,"excerpt":"已在支付渠道确认退款已发起，预计 1～3 个工作日到账。"}',
   now() - interval '27 hours'),
  (5, 2, 'status_changed', '{"from":"in_progress","to":"resolved","note":"退款已发起，等待到账"}',
   now() - interval '26 hours'),
  -- 工单 6：open → in_progress → resolved → closed
  (6, 1, 'created', '{"priority":"low"}', now() - interval '5 days'),
  (6, 4, 'claimed', '{"assignee_id":4}', now() - interval '5 days' + interval '1 hour'),
  (6, 4, 'status_changed', '{"from":"in_progress","to":"resolved","note":"已按客户要求修改地址"}',
   now() - interval '4 days' - interval '2 hours'),
  (6, 4, 'status_changed', '{"from":"resolved","to":"closed","note":""}', now() - interval '4 days');

-- 显式指定了主键，序列要推到最大 ID 之后，否则后续插入会主键冲突。
SELECT setval(pg_get_serial_sequence('users', 'id'),           (SELECT max(id) FROM users));
SELECT setval(pg_get_serial_sequence('tickets', 'id'),         (SELECT max(id) FROM tickets));
SELECT setval(pg_get_serial_sequence('ticket_comments', 'id'), (SELECT max(id) FROM ticket_comments));
SELECT setval(pg_get_serial_sequence('ticket_events', 'id'),   (SELECT max(id) FROM ticket_events));
