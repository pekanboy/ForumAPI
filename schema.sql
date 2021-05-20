DROP SCHEMA IF EXISTS forum CASCADE;
CREATE SCHEMA forum;

create extension if not exists citext with schema forum;

CREATE TABLE forum.user
(
    id       BIGSERIAL PRIMARY KEY,
    nickname forum.citext UNIQUE NOT NULL,
    fullname TEXT                NOT NULL,
    about    TEXT                NOT NULL,
    email    TEXT UNIQUE         NOT NULL
);

CREATE TABLE forum.forum
(
    id      BIGSERIAL PRIMARY KEY,
    title   TEXT         NOT NULL,
    "user"  forum.citext NOT NULL,
    slug    TEXT UNIQUE  NOT NULL,
    posts   BIGINT       NOT NULL,
    threads BIGINT       NOT NULL,
    FOREIGN KEY ("user")
        REFERENCES forum.user (nickname)
);

CREATE TABLE forum.thread
(
    id      BIGSERIAL PRIMARY KEY,
    title   TEXT                     NOT NULL,
    author  forum.citext             NOT NULL,
    forum   TEXT                     NOT NULL,
    message TEXT                     NOT NULL,
    votes   BIGINT                   NOT NULL,
    slug    TEXT UNIQUE,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    FOREIGN KEY (author)
        REFERENCES forum.user (nickname),
    FOREIGN KEY (forum)
        REFERENCES forum.forum (slug)
);

CREATE OR REPLACE FUNCTION forum.forum_threads_inc()
    RETURNS TRIGGER AS
$$
BEGIN
    UPDATE forum.forum SET threads = threads + 1 WHERE slug = NEW.forum;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS forum_thread ON forum.thread;
CREATE TRIGGER forum_thread
    AFTER INSERT
    ON forum.thread
    FOR EACH ROW
EXECUTE PROCEDURE forum.forum_threads_inc();

CREATE TABLE forum.post
(
    id       BIGSERIAL PRIMARY KEY,
    parent   BIGINT                   NOT NULL,
    author   forum.citext             NOT NULL,
    message  TEXT                     NOT NULL,
    isEdited BOOLEAN                  NOT NULL,
    forum    TEXT                     NOT NULL,
    thread   BIGINT                   NOT NULL,
    created  TIMESTAMP WITH TIME ZONE NOT NULL,
    FOREIGN KEY (author)
        REFERENCES forum.user (nickname),
    FOREIGN KEY (forum)
        REFERENCES forum.forum (slug),
    FOREIGN KEY (thread)
        REFERENCES forum.thread (id)
);

CREATE OR REPLACE FUNCTION forum.forum_posts_inc()
    RETURNS TRIGGER AS
$$
BEGIN
    UPDATE forum.forum SET posts = posts + 1 WHERE slug = NEW.forum;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS forum_post ON forum.post;
CREATE TRIGGER forum_post
    AFTER INSERT
    ON forum.post
    FOR EACH ROW
EXECUTE PROCEDURE forum.forum_posts_inc();