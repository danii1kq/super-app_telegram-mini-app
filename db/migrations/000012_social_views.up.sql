CREATE TABLE social_post_views (
    post_id bigint NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, user_id)
);
