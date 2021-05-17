DROP SCHEMA IF EXISTS forum CASCADE;
CREATE SCHEMA forum;

CREATE TABLE forum.user
(
    id       BIGSERIAL PRIMARY KEY,
    nickname TEXT UNIQUE NOT NULL,
    fullname TEXT        NOT NULL,
    about    TEXT        NOT NULL,
    email    TEXT UNIQUE NOT NULL
);

CREATE TABLE forum.forum
(
    id       BIGSERIAL PRIMARY KEY,
    title    TEXT        NOT NULL,
    "user"   TEXT        NOT NULL,
    slug     TEXT UNIQUE NOT NULL,
    posts    BIGINT      NOT NULL,
    threads  BIGINT      NOT NULL,
    FOREIGN KEY("user")
        REFERENCES forum.user(nickname)
);

-- Навесить тригеры на увеличение posts & threads