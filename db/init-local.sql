-- 不使用 Docker 时，用本机 PostgreSQL 16 的超级用户（安装时创建的 postgres）执行一次：
--   psql -U postgres -h 127.0.0.1 -f db/init-local.sql
-- 账号密码与 .env.example 一致，仅用于本地开发。
CREATE USER ticketdesk WITH PASSWORD 'ticketdesk';
CREATE DATABASE ticketdesk OWNER ticketdesk ENCODING 'UTF8';
CREATE DATABASE ticketdesk_test OWNER ticketdesk ENCODING 'UTF8';
