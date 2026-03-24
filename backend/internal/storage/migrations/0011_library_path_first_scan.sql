-- first_library_scan_pending: 1 = 该库根尚未完成「首次入库扫描」；新增路径默认为 1，成功扫过一次后清零。
-- 存量行迁移后为 0，与升级前行为一致（不触发扩展导入逻辑）。
ALTER TABLE library_paths ADD COLUMN first_library_scan_pending INTEGER NOT NULL DEFAULT 0;
