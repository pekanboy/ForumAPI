ALTER USER postgres WITH ENCRYPTED PASSWORD 'admin';
DROP SCHEMA IF EXISTS forum CASCADE;
CREATE SCHEMA forum;

create extension if not exists citext;

-- FUNCTIONS

CREATE OR REPLACE FUNCTION forum.forum_posts_inc()
    RETURNS TRIGGER AS
$$
DECLARE
    parentPath BIGINT[];
BEGIN
    IF (NEW.parent IS NULL) THEN
        NEW.path := array_append(new.path, new.id);
    ELSE
        SELECT path FROM forum.post WHERE id = new.parent INTO parentPath;
        NEW.path := NEW.path || parentPath || new.id;
    end if;

    UPDATE forum.forum SET posts = posts + 1 WHERE slug = NEW.forum;

    INSERT INTO forum.forum_users(forum, nickname, fullname, about, email)
    SELECT NEW.forum, nickname, fullname, about, email
    FROM forum.user
    WHERE nickname = NEW.author
    ON CONFLICT DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION forum.thread_votes_inc()
    RETURNS TRIGGER AS
$$
BEGIN
    UPDATE forum.thread SET votes = votes + NEW.voice WHERE id = NEW.thread;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION forum.thread_votes_inc_2()
    RETURNS TRIGGER AS
$$
BEGIN
    UPDATE forum.thread SET votes = votes + NEW.voice - OLD.voice WHERE id = NEW.thread;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION forum.forum_threads_inc()
    RETURNS TRIGGER AS
$$
BEGIN
    UPDATE forum.forum SET threads = threads + 1 WHERE slug = NEW.forum;

    INSERT INTO forum.forum_users(forum, nickname, fullname, about, email)
    SELECT NEW.forum, nickname, fullname, about, email
    FROM forum.user
    WHERE nickname = NEW.author
    ON CONFLICT DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

--  USER

CREATE UNLOGGED TABLE forum.user
(
    id       BIGSERIAL PRIMARY KEY,
    nickname citext collate "POSIX" UNIQUE NOT NULL,
    fullname TEXT                          NOT NULL,
    about    TEXT,
    email    citext UNIQUE                 NOT NULL
);

-- FORUM

CREATE UNLOGGED TABLE forum.forum
(
    id      BIGSERIAL PRIMARY KEY,
    title   TEXT          NOT NULL,
    "user"  citext        NOT NULL,
    slug    citext UNIQUE NOT NULL,
    posts   BIGINT        NOT NULL DEFAULT 0,
    threads BIGINT        NOT NULL DEFAULT 0,
    FOREIGN KEY ("user")
        REFERENCES forum.user (nickname)
);

-- THREAD

CREATE UNLOGGED TABLE forum.thread
(
    id      BIGSERIAL PRIMARY KEY,
    title   TEXT                     NOT NULL,
    author  citext                   NOT NULL,
    forum   citext                   NOT NULL,
    message TEXT                     NOT NULL,
    votes   BIGINT                   NOT NULL DEFAULT 0,
    slug    citext UNIQUE,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    FOREIGN KEY (author)
        REFERENCES forum.user (nickname),
    FOREIGN KEY (forum)
        REFERENCES forum.forum (slug)
);

DROP TRIGGER IF EXISTS forum_thread ON forum.thread;
CREATE TRIGGER forum_thread
    AFTER INSERT
    ON forum.thread
    FOR EACH ROW
EXECUTE PROCEDURE forum.forum_threads_inc();

-- POST

CREATE UNLOGGED TABLE forum.post
(
    id       BIGSERIAL PRIMARY KEY,
    parent   BIGINT                   NOT NULL,
    author   citext                   NOT NULL,
    message  TEXT                     NOT NULL,
    isEdited BOOLEAN                  NOT NULL DEFAULT false,
    forum    citext                   NOT NULL,
    thread   BIGINT                   NOT NULL,
    created  TIMESTAMP WITH TIME ZONE NOT NULL,
    path     BIGINT[]                 NOT NULL DEFAULT ARRAY []::INTEGER[],
    FOREIGN KEY (author)
        REFERENCES forum.user (nickname),
    FOREIGN KEY (forum)
        REFERENCES forum.forum (slug),
    FOREIGN KEY (thread)
        REFERENCES forum.thread (id)
);

DROP TRIGGER IF EXISTS forum_post ON forum.post;
CREATE TRIGGER forum_post
    BEFORE INSERT
    ON forum.post
    FOR EACH ROW
EXECUTE PROCEDURE forum.forum_posts_inc();

-- VOTE

CREATE UNLOGGED TABLE forum.vote
(
    id       BIGSERIAL NOT NULL,
    thread   bigint NOT NULL,
    nickname citext NOT NULL,
    voice    BIGINT NOT NULL,
    FOREIGN KEY (thread)
        REFERENCES forum.thread (id),
    FOREIGN KEY (nickname)
        REFERENCES forum.user (nickname),
    PRIMARY KEY (thread, nickname)
);

DROP TRIGGER IF EXISTS forum_vote ON forum.vote;
CREATE TRIGGER forum_vote
    AFTER INSERT
    ON forum.vote
    FOR EACH ROW
EXECUTE PROCEDURE forum.thread_votes_inc();

DROP TRIGGER IF EXISTS forum_vote_2 ON forum.vote;
CREATE TRIGGER forum_vote_2
    AFTER UPDATE
    ON forum.vote
    FOR EACH ROW
EXECUTE PROCEDURE forum.thread_votes_inc_2();

-- FORUM USERS

CREATE UNLOGGED TABLE forum.forum_users
(
    forum    citext                 NOT NULL,
    nickname citext collate "POSIX" NOT NULL,
    fullname TEXT                   NOT NULL,
    about    TEXT,
    email    citext          NOT NULL,
    FOREIGN KEY (forum)
        REFERENCES forum.forum (slug),
    FOREIGN KEY (nickname)
        REFERENCES forum.user (nickname),
    PRIMARY KEY (nickname, forum)
);
