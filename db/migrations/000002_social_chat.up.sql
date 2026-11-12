-- Social core: posts, comments, reactions, follows, communities, blocks,
-- reports and moderation. Owner: backend/internal/core/social.

CREATE TABLE social_posts (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    community_id  bigint,
    audience      text        NOT NULL DEFAULT 'public'
                  CHECK (audience IN ('public', 'followers', 'community')),
    body_public   text,
    body_sealed   bytea,
    key_id        text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    -- Public text lives in body_public; restricted text in the sealed pair.
    CHECK (
        (audience = 'public' AND body_public IS NOT NULL)
        OR (audience <> 'public' AND body_sealed IS NOT NULL AND key_id IS NOT NULL)
    )
);

CREATE INDEX social_posts_author_idx ON social_posts (author_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX social_posts_community_idx ON social_posts (community_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND community_id IS NOT NULL;

CREATE TABLE social_comments (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id      bigint      NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    author_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body_public  text,
    body_sealed  bytea,
    key_id       text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);

CREATE INDEX social_comments_post_idx ON social_comments (post_id, created_at, id)
    WHERE deleted_at IS NULL;

CREATE TABLE social_reactions (
    post_id     bigint      NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    user_id     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        text        NOT NULL CHECK (kind IN ('like', 'heart', 'fire')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id, kind)
);

CREATE TABLE social_follows (
    follower_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    author_id   bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, author_id),
    CHECK (follower_id <> author_id)
);

CREATE INDEX social_follows_author_idx ON social_follows (author_id);

CREATE TABLE social_communities (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name_sealed   bytea       NOT NULL,
    key_id        text        NOT NULL,
    invite_hash   text        UNIQUE,
    state         text        NOT NULL DEFAULT 'active'
                  CHECK (state IN ('active', 'archived')),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE social_members (
    community_id bigint      NOT NULL REFERENCES social_communities(id) ON DELETE CASCADE,
    user_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         text        NOT NULL DEFAULT 'member'
                 CHECK (role IN ('owner', 'moderator', 'member')),
    state        text        NOT NULL DEFAULT 'active'
                 CHECK (state IN ('invited', 'active', 'left', 'removed')),
    joined_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (community_id, user_id)
);

CREATE INDEX social_members_user_idx ON social_members (user_id) WHERE state = 'active';

ALTER TABLE social_posts
    ADD CONSTRAINT social_posts_community_fk
    FOREIGN KEY (community_id) REFERENCES social_communities(id) ON DELETE CASCADE;

CREATE TABLE social_blocks (
    blocker_id  bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id  bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE TABLE social_reports (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reporter_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_kind text        NOT NULL CHECK (target_kind IN ('post', 'comment', 'message', 'user')),
    target_id   bigint      NOT NULL,
    reason      text        NOT NULL,
    state       text        NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'resolved', 'rejected')),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE social_moderation_actions (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    moderator_id bigint     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_kind text        NOT NULL,
    target_id   bigint      NOT NULL,
    action      text        NOT NULL CHECK (action IN ('hide', 'restore', 'warn', 'ban')),
    note        text,
    created_at  timestamptz NOT NULL DEFAULT now()
);
