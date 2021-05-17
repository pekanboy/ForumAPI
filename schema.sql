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
    id      BIGSERIAL PRIMARY KEY,
    title   TEXT        NOT NULL,
    "user"  TEXT        NOT NULL,
    slug    TEXT UNIQUE NOT NULL,
    posts   BIGINT      NOT NULL,
    threads BIGINT      NOT NULL,
    FOREIGN KEY ("user")
        REFERENCES forum.user (nickname)
);

CREATE TABLE forum.thread
(
    id      BIGSERIAL PRIMARY KEY,
    title   TEXT                     NOT NULL,
    author  TEXT                     NOT NULL,
    forum   TEXT                     NOT NULL,
    message TEXT                     NOT NULL,
    votes   BIGINT                   NOT NULL,
    slug    TEXT                     NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    FOREIGN KEY (author)
        REFERENCES forum.user (nickname),
    FOREIGN KEY (forum)
        REFERENCES forum.forum (slug)
);

CREATE OR REPLACE FUNCTION forum.forum_threads_inc()
    RETURNS TRIGGER AS $$
    BEGIN
        UPDATE forum.forum SET threads = threads + 1 WHERE slug = NEW.forum;

        RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS forum_thread ON forum.thread;
CREATE TRIGGER forum_thread AFTER INSERT ON forum.thread
    FOR EACH ROW EXECUTE PROCEDURE forum.forum_threads_inc();