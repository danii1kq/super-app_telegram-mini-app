package social

// Every post surface uses the same access rule before private text is opened.
// $1 is the viewer and p is the post being selected.
const visiblePostSQL = `p.deleted_at IS NULL
 AND NOT EXISTS (SELECT 1 FROM social_blocks b
   WHERE (b.blocker_id = p.author_id AND b.blocked_id = $1)
      OR (b.blocker_id = $1 AND b.blocked_id = p.author_id))
 AND (
   (p.community_id IS NULL AND (
     p.author_id = $1 OR p.audience = 'public'
     OR (p.audience = 'followers' AND EXISTS (SELECT 1 FROM social_follows f
       WHERE f.follower_id = $1 AND f.author_id = p.author_id))))
   OR (p.audience = 'community' AND EXISTS (
     SELECT 1 FROM social_members m JOIN social_communities c ON c.id = m.community_id
     WHERE m.community_id = p.community_id AND m.user_id = $1
       AND m.state = 'active' AND c.state = 'active'))
 )`
