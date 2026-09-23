-- Tugas mandiri modul 6: sistem komentar dengan RBAC.

CREATE TABLE IF NOT EXISTS articles (
    id         SERIAL PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    content    TEXT         NOT NULL,
    author_id  INTEGER      NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS comments (
    id         SERIAL PRIMARY KEY,
    article_id INTEGER     NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    author_id  INTEGER     NOT NULL REFERENCES users(id),
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS comments_article_id_idx ON comments (article_id);

-- User uji untuk skenario pengujian (password semuanya: password123).
INSERT INTO users (username, email, password, role_id, is_active) VALUES
    ('admin1',  'admin@test.com',  '$2a$12$RfjnZctgtMeOLefCymwCZuOOr/kxgcI.TvdxgcThA11OA3qIgZYi2',
     (SELECT id FROM roles WHERE name = 'admin'), TRUE),
    ('editor1', 'editor@test.com', '$2a$12$RfjnZctgtMeOLefCymwCZuOOr/kxgcI.TvdxgcThA11OA3qIgZYi2',
     (SELECT id FROM roles WHERE name = 'editor'), TRUE),
    ('viewer1', 'viewer@test.com', '$2a$12$RfjnZctgtMeOLefCymwCZuOOr/kxgcI.TvdxgcThA11OA3qIgZYi2',
     (SELECT id FROM roles WHERE name = 'viewer'), TRUE)
ON CONFLICT DO NOTHING;

-- Satu artikel contoh milik editor1, dipakai untuk menguji comments.
INSERT INTO articles (title, content, author_id)
SELECT 'Artikel Pertama', 'Isi artikel pertama untuk uji komentar.', id
FROM users WHERE username = 'editor1'
AND NOT EXISTS (SELECT 1 FROM articles);

-- Satu komentar contoh milik viewer1 di artikel tersebut,
-- dipakai untuk menguji update/delete komentar milik sendiri vs milik orang lain.
INSERT INTO comments (article_id, author_id, content)
SELECT a.id, u.id, 'Komentar awal dari viewer1.'
FROM articles a, users u
WHERE u.username = 'viewer1' AND a.author_id <> u.id
AND NOT EXISTS (SELECT 1 FROM comments);
