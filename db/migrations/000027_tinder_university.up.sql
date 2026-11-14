-- SruTinder: the profile field is a university, not a city.
-- Owner: backend/internal/apps/tinder.
--
-- The audience is students in one city, so "which city" carries no signal:
-- everyone answers Almaty. The university is the field that actually tells
-- someone who they are looking at.

ALTER TABLE tinder_profiles RENAME COLUMN city TO university;
