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