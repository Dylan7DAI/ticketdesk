-- 集成测试专用库。开发库 ticketdesk 由 POSTGRES_DB 创建。
-- 用 .sql 而不是 .sh：.sh 文件带 CRLF 换行时容器内无法执行。
CREATE DATABASE ticketdesk_test;
