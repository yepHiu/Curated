ALTER TABLE wishlist_movie_links ADD COLUMN manual INTEGER NOT NULL DEFAULT 0;
-- 已有显式排除保持人工来源，后续自动对账不得撤销。
UPDATE wishlist_movie_links SET manual=1 WHERE excluded=1;
