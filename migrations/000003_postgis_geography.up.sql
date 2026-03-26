CREATE EXTENSION IF NOT EXISTS postgis;

ALTER TABLE incidents ADD COLUMN IF NOT EXISTS geog geography(Point, 4326);

UPDATE incidents SET geog = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography WHERE geog IS NULL;

CREATE INDEX IF NOT EXISTS idx_incidents_geog ON incidents USING GIST (geog);

