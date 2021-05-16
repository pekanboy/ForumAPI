drop table if exists user_test;

CREATE TABLE user_test
(
    id       BIGSERIAL PRIMARY KEY,
    nickname TEXT UNIQUE NOT NULL,
    fullname TEXT        NOT NULL,
    about    TEXT        NOT NULL,
    email    TEXT UNIQUE NOT NULL
)